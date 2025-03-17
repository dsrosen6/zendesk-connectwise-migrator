package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"slices"
	"strings"
	"time"
)

type orgCheckerModel struct {
	migrationClient *migration.Client
	tags            []tagDetails
	orgs            *allOrgs
	status          status
	done            bool
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
	zendeskOrg *zendesk.Organization
	psaOrg     *psa.Company
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
		orgs:            &allOrgs{},
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
			cmd = m.constructOutput()
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

	s += fmt.Sprintf("\n\nChecked: %d/%d\n"+
		"With Tickets: %d\n"+
		"In PSA/With Tickets: %d/%d\n"+
		"Not in PSA: %d\n"+
		"Errored: %d\n",
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
			q := &zendesk.SearchQuery{}
			q.Tags = []string{tag.name}

			slog.Info("getting all orgs from zendesk for tag group", "tag", tag.name)

			orgs, err := m.migrationClient.ZendeskClient.GetOrganizationsWithQuery(ctx, *q)
			if err != nil {
				return apiErrMsg{err}
			}

			for _, org := range orgs {
				md := &orgMigrationDetails{
					zendeskOrg: &org,
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

		tickets, err := m.migrationClient.ZendeskClient.GetTicketsWithQuery(ctx, *q, 20, true)
		if err != nil {
			m.orgs.erroredOrgs = append(m.orgs.erroredOrgs, erroredOrg{org: org, err: err})
		}

		if len(tickets) > 0 {
			m.orgs.withTickets = append(m.orgs.withTickets, org)
			if m.orgInPsa(org) {
				slog.Debug("org in PSA", "orgName", org.zendeskOrg.Name)
				if err := m.updateCompanyFieldValue(org); err != nil {
					m.orgs.erroredOrgs = append(m.orgs.erroredOrgs, erroredOrg{org: org, err: err})
					return nil
				}

				m.orgs.inPsa = append(m.orgs.inPsa, org)
			} else {
				slog.Debug("org not in PSA", "orgName", org.zendeskOrg.Name)
				m.orgs.notInPsa = append(m.orgs.notInPsa, org)
			}
		}

		m.orgs.checked = append(m.orgs.checked, org)
		return nil
	}
}

func (m *orgCheckerModel) orgInPsa(org *orgMigrationDetails) bool {
	var err error
	org.psaOrg, err = m.migrationClient.MatchZdOrgToCwCompany(ctx, org.zendeskOrg)
	return err == nil
}

func (m *orgCheckerModel) updateCompanyFieldValue(org *orgMigrationDetails) error {
	if org.zendeskOrg.OrganizationFields.PSACompanyId != 0 {
		slog.Debug("zendesk org already has PSA company id field", "orgName", org.zendeskOrg.Name, "psaCompanyId", org.zendeskOrg.OrganizationFields.PSACompanyId)
		return nil
	}

	if org.psaOrg.Id != 0 {
		org.zendeskOrg.OrganizationFields.PSACompanyId = int64(org.psaOrg.Id)

		var err error
		org.zendeskOrg, err = m.migrationClient.ZendeskClient.UpdateOrganization(ctx, org.zendeskOrg)
		if err != nil {
			return fmt.Errorf("updating organization with PSA company id: %w", err)
		}

		slog.Info("updated zendesk organization with PSA company id", "orgName", org.zendeskOrg.Name, "psaCompanyId", org.psaOrg.Id)
	}

	return nil
}

func switchStatus(s status) tea.Cmd {
	return func() tea.Msg {
		return switchStatusMsg(s)
	}
}

func (m *orgCheckerModel) constructOutput() tea.Cmd {
	return func() tea.Msg {
		var notInPsa, errored []string

		for _, org := range m.orgs.notInPsa {
			notInPsa = append(notInPsa, org.zendeskOrg.Name)
		}

		for _, org := range m.orgs.erroredOrgs {
			summary := fmt.Sprintf("%s\n%s", org.org.zendeskOrg.Name, org.err)
			errored = append(errored, summary)
		}

		slices.Sort(notInPsa)
		slices.Sort(errored)

		var output string
		if len(notInPsa) > 0 {
			output += lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				Bold(true).
				Render("Zendesk Organizations Not in PSA")
			output += fmt.Sprintf("\n%s\n", strings.Join(notInPsa, "\n"))
		}

		if len(errored) > 0 {
			output += lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				Bold(true).
				Render("Errors")
			output += fmt.Sprintf("\n%s\n", strings.Join(errored, "\n"))
		}

		return updateResultsMsg{title: "Results", body: output}
	}
}
