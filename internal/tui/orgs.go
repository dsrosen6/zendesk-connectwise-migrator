package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	zendesk2 "github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"slices"
	"strings"
	"time"
)

type orgCheckerModel struct {
	tags            []tagDetails
	orgs            allOrgs
	migrationClient *migration.Client
	status          status
	orgsNotInPsa    []zendesk2.Organization
	done            bool
	viewport        viewport.Model
}

type tagDetails struct {
	name      string
	startDate time.Time
	endDate   time.Time
}

type allOrgs struct {
	master      []*orgMigrationDetails
	checked     []*orgMigrationDetails
	withTickets []*orgMigrationDetails
	inPsa       []*orgMigrationDetails
	notInPsa    []*orgMigrationDetails
	erroredOrgs []erroredOrg
}

type orgMigrationDetails struct {
	tag        *tagDetails
	zendeskOrg zendesk2.Organization
	psaOrg     psa.Company
}

type erroredOrg struct {
	org *orgMigrationDetails
	err error
}

type switchStatusMsg string

type status string

const (
	gettingTags        status = "gettingTags"
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
	return m.getTagDetails(m.migrationClient.Cfg.Zendesk.TagsToMigrate)
}

func (m *orgCheckerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.status == done && msg.String() == "m" {
			return m, switchModel(newMainMenuModel(m.migrationClient))
		}

	case switchStatusMsg:
		switch msg {
		case switchStatusMsg(gettingZendeskOrgs):
			slog.Debug("org checker: got tags", "tags", m.tags)
			m.status = gettingZendeskOrgs
			return m, m.getOrgs()

		case switchStatusMsg(comparingOrgs):
			slog.Debug("org checker: got orgs", "total", len(m.orgs.master))
			m.status = comparingOrgs
			var checkOrgCmds []tea.Cmd
			for _, org := range m.orgs.master {
				checkOrgCmds = append(checkOrgCmds, m.checkOrg(org))
			}
			return m, tea.Sequence(checkOrgCmds...)

		case switchStatusMsg(done):
			slog.Debug("org checker: done checking orgs")
			m.status = done
			cmd = constructOutput(m.orgs.notInPsa)
			cmds = append(cmds, cmd)
		}
	}

	if m.status == comparingOrgs && !m.done {
		if len(m.orgs.master) == len(m.orgs.checked) {
			cmd = switchStatus(done)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *orgCheckerModel) View() string {
	var s, st string
	switch m.status {
	case gettingTags:
		st = runSpinner("Getting Zendesk tags")
	case gettingZendeskOrgs:
		st = runSpinner("Getting Zendesk orgs")
	case comparingOrgs:
		st = runSpinner("Checking for PSA orgs")
	case done:
		st = "Done - press 'm' to return to the main menu"
	}

	s += st

	s += fmt.Sprintf("\n\nChecked: %d/%d\nWith Tickets: %d\nIn PSA/With Tickets: %d/%d\nNot in PSA: %d\nErrored: %d\n",
		len(m.orgs.checked), len(m.orgs.master), len(m.orgs.withTickets), len(m.orgs.inPsa), len(m.orgs.withTickets), len(m.orgs.notInPsa), len(m.orgs.erroredOrgs))

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
		return switchStatusMsg(gettingZendeskOrgs)
	}
}

func (m *orgCheckerModel) getOrgs() tea.Cmd {
	return func() tea.Msg {
		slog.Debug("getting orgs for tags", "tags", m.migrationClient.Cfg.Zendesk.TagsToMigrate)
		for _, tag := range m.tags {
			slog.Debug("getting orgs for tag", "tag", tag.name)
			q := &zendesk2.SearchQuery{}
			q.Tags = []string{tag.name}

			slog.Info("getting all orgs from zendesk for tag group", "tag", tag.name)

			orgs, err := m.migrationClient.ZendeskClient.GetOrganizationsWithQuery(ctx, *q)
			if err != nil {
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
		q := &zendesk2.SearchQuery{}
		if org.tag.startDate != (time.Time{}) {
			q.TicketCreatedAfter = org.tag.startDate
		}

		if org.tag.endDate != (time.Time{}) {
			q.TicketCreatedBefore = org.tag.endDate
		}

		q.TicketsOrganizationId = org.zendeskOrg.Id

		tickets, err := m.migrationClient.ZendeskClient.GetTicketsWithQuery(ctx, *q, 20, true)
		if err != nil {
			m.orgs.erroredOrgs = append(m.orgs.erroredOrgs, erroredOrg{org: org, err: err})
		}

		if len(tickets) > 0 {
			m.orgs.withTickets = append(m.orgs.withTickets, org)
			if m.orgInPsa(org) {
				m.orgs.inPsa = append(m.orgs.inPsa, org)
			} else {
				m.orgs.notInPsa = append(m.orgs.notInPsa, org)
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

func switchStatus(s status) tea.Cmd {
	return func() tea.Msg {
		return switchStatusMsg(s)
	}
}

func constructOutput(orgs []*orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		var names []string
		if len(orgs) == 0 {
			return updateResultsMsg{title: "Orgs Not in PSA", body: "All Orgs are in the PSA!"}
		}

		for _, org := range orgs {
			names = append(names, org.zendeskOrg.Name)
		}

		slices.Sort(names)
		output := strings.Join(names, "\n")
		return updateResultsMsg{title: "Orgs Not in PSA", body: output}
	}
}
