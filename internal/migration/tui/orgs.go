package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/apis/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/apis/zendesk"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"log/slog"
	"time"
)

type orgCheckerModel struct {
	tags            []tagDetails
	orgs            allOrgs
	migrationClient *migration.Client
	status          status
	orgsNotInPsa    []zendesk.Organization
	done            bool
}

type tagDetails struct {
	name      string
	startDate time.Time
	endDate   time.Time
}

type allOrgs struct {
	master        []*orgMigrationDetails
	checked       []*orgMigrationDetails
	withTickets   []*orgMigrationDetails
	inPsa         []*orgMigrationDetails
	notInPsa      []*orgMigrationDetails
	notInPsaNames string
	erroredOrgs   []erroredOrg
}

type orgMigrationDetails struct {
	tag        *tagDetails
	zendeskOrg zendesk.Organization
	psaOrg     psa.Company
}

type erroredOrg struct {
	org *orgMigrationDetails
	err error
}

type switchStatusMsg string

type status string

const (
	gettingZendeskOrgs status = "gettingZendeskOrgs"
	comparingOrgs      status = "comparingOrgs"
	done               status = "done"
)

func newOrgCheckerModel(mc *migration.Client) *orgCheckerModel {
	return &orgCheckerModel{
		migrationClient: mc,
		status:          gettingZendeskOrgs,
	}
}

func (m *orgCheckerModel) Init() tea.Cmd {
	return m.getOrgs()
}

func (m *orgCheckerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.status == done {
			return m, switchModel(newMainMenuModel(m.migrationClient))
		}

	case switchStatusMsg:
		switch msg {
		case switchStatusMsg(comparingOrgs):
			slog.Debug("got orgs", "total", len(m.orgs.master))
			m.status = comparingOrgs
			var checkOrgCmds []tea.Cmd
			for _, org := range m.orgs.master {
				checkOrgCmds = append(checkOrgCmds, m.checkOrg(org))
			}
			return m, tea.Sequence(checkOrgCmds...)
		case switchStatusMsg(done):
			m.done = true
		}
	}

	if m.status == comparingOrgs && !m.done {
		if len(m.orgs.master) == len(m.orgs.checked) {
			slog.Debug("done")
			m.status = done
		}
	}

	return m, nil
}

func (m *orgCheckerModel) View() string {
	var s, st string
	switch m.status {
	case gettingZendeskOrgs:
		st = runSpinner("Getting Zendesk orgs")
	case comparingOrgs:
		st = runSpinner("Checking orgs")
	case done:
		st = " Done checking orgs"
	}

	s += st

	s += fmt.Sprintf(" Checked: %d/%d\n With Tickets: %d\n In PSA/With Tickets: %d/%d\n",
		len(m.orgs.checked), len(m.orgs.master), len(m.orgs.withTickets), len(m.orgs.inPsa), len(m.orgs.withTickets))

	if m.orgs.notInPsaNames != "" {
		s += fmt.Sprintf("\nZendesk Orgs not in PSA:\n%s\n", m.orgs.notInPsaNames)
	}

	return s
}

func (m *orgCheckerModel) getTagDetails(tags []migration.TagDetails) tea.Cmd {
	return func() tea.Msg {
		for _, tag := range tags {
			tm := &timeConversionDetails{
				startString:   tag.StartDate,
				endString:     tag.EndDate,
				startFallback: m.migrationClient.Cfg.Zendesk.MasterStartDate,
				endFallback:   m.migrationClient.Cfg.Zendesk.MasterEndDate,
			}

			start, end, err := convertDateStringsToTimeTime(tm)
			if err != nil {
				return timeConvertErrMsg{err}
			}

			td := tagDetails{
				name:      tag.Name,
				startDate: start,
				endDate:   end,
			}

			m.tags = append(m.tags, td)
		}
		return nil
	}
}
func (m *orgCheckerModel) getOrgs() tea.Cmd {
	slog.Debug("starting getOrgs")
	return func() tea.Msg {
		for _, tag := range m.tags {
			q := &zendesk.SearchQuery{}
			q.Tags = []string{tag.name}

			slog.Info("getting all orgs from zendesk for tag group", "tag", tag.name)

			orgs, err := m.migrationClient.ZendeskClient.GetOrganizationsWithQuery(ctx, *q)
			if err != nil {
				slog.Error("error getting orgs", "err", err)
				return apiErrMsg{err}
			}

			for _, org := range orgs {
				md := &orgMigrationDetails{
					zendeskOrg: org,
					tag:        &tag,
				}
				m.orgs.master = append(m.orgs.master, md)
			}
		}
		return switchStatusMsg(comparingOrgs)
	}
}

func (m *orgCheckerModel) checkOrg(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		q := &zendesk.SearchQuery{}
		if org.tag.startDate != (time.Time{}) {
			q.TicketCreatedAfter = org.tag.startDate
		}

		if org.tag.endDate != (time.Time{}) {
			q.TicketCreatedBefore = org.tag.endDate
		}

		q.TicketsOrganizationId = org.zendeskOrg.Id

		tickets, err := m.migrationClient.ZendeskClient.GetTicketsWithQuery(ctx, *q)
		if err != nil {
			m.orgs.erroredOrgs = append(m.orgs.erroredOrgs, erroredOrg{org: org, err: err})
		}

		if len(tickets) > 0 {
			m.orgs.withTickets = append(m.orgs.withTickets, org)
			if m.orgInPsa(org) {
				m.orgs.inPsa = append(m.orgs.inPsa, org)
			} else {
				m.orgs.notInPsa = append(m.orgs.inPsa)
				m.orgs.notInPsaNames += fmt.Sprintf("%s\n", org.zendeskOrg.Name)
			}
		}

		m.orgs.checked = append(m.orgs.checked, org)
		return nil
	}
}

func (m *orgCheckerModel) orgInPsa(org *orgMigrationDetails) bool {
	_, err := m.migrationClient.MatchZdOrgToCwCompany(ctx, org.zendeskOrg)
	return err == nil
}
