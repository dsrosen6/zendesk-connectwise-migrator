package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
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
	toMigrate   []*orgMigrationDetails
	erroredOrgs []erroredOrg
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
			cmds = append(cmds, m.constructOutput())
			cmds = append(cmds, sendOrgsCmd(m.orgs.toMigrate))
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
		"Errored: %d\n",
		len(m.orgs.checked), len(m.orgs.master), len(m.orgs.erroredOrgs))

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
		slog.Info("getting orgs for tags", "tags", m.migrationClient.Cfg.Zendesk.TagsToMigrate)
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
			slog.Error("error getting tickets for org", "orgName", org.zendeskOrg.Name, "error", err)
			m.orgs.erroredOrgs = append(m.orgs.erroredOrgs, erroredOrg{org: org, err: err})
			return nil
		}

		if len(tickets) > 0 {
			org.hasTickets = true
		} else {
			// We only care about orgs with tickets - no need to check further
			return nil
		}

		org.psaOrg, err = m.migrationClient.MatchZdOrgToCwCompany(ctx, org.zendeskOrg)
		if err != nil {
			slog.Warn("org is not in PSA", "orgName", org.zendeskOrg.Name)
			return nil
		}

		if err := m.updateCompanyFieldValue(org); err != nil {
			slog.Error("error updating company field value in zendesk", "error", err)
			m.orgs.erroredOrgs = append(m.orgs.erroredOrgs, erroredOrg{org: org, err: err})
			return nil
		}

		if org.psaOrg != nil && org.zendeskOrg.OrganizationFields.PSACompanyId == int64(org.psaOrg.Id) {
			org.ready = true
			m.orgs.toMigrate = append(m.orgs.toMigrate, org)
		}

		m.orgs.checked = append(m.orgs.checked, org)
		return nil

	}
}

func (m *orgCheckerModel) updateCompanyFieldValue(org *orgMigrationDetails) error {
	if org.zendeskOrg.OrganizationFields.PSACompanyId == int64(org.psaOrg.Id) {
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
		return nil
	} else {
		slog.Warn("org psa id is 0 - cannot update psa_company field in zendesk", "orgName", org.zendeskOrg.Name)
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
		var tagNames, withTickets, notInPsa, notReady, ready, errored []string

		for _, tag := range m.tags {
			tagNames = append(tagNames, tag.name)
		}

		for _, org := range m.orgs.checked {
			if org.hasTickets {
				withTickets = append(withTickets, org.zendeskOrg.Name)
			}

			if org.psaOrg == nil {
				notInPsa = append(notInPsa, org.zendeskOrg.Name)
			}

			if org.ready {
				ready = append(ready, org.zendeskOrg.Name)
			} else {
				notReady = append(notReady, org.zendeskOrg.Name)
			}
		}

		for _, org := range m.orgs.erroredOrgs {
			summary := fmt.Sprintf("%s\n%s", org.org.zendeskOrg.Name, org.err)
			errored = append(errored, summary)
		}

		slices.Sort(tagNames)
		slices.Sort(withTickets)
		slices.Sort(notInPsa)
		slices.Sort(notReady)
		slices.Sort(ready)
		slices.Sort(errored)

		var output string

		output += lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			Bold(true).
			Render("Statistics")

		output += fmt.Sprintf("\nTags Checked: %s\n"+
			"With Tickets: %d\n"+
			"Not in PSA: %d\n"+
			"Ready for User Migration: %d/%d\n"+
			"Errored: %d\n\n",
			strings.Join(tagNames, ", "),
			len(withTickets),
			len(notInPsa),
			len(ready), len(withTickets),
			len(errored))

		if len(withTickets) == len(ready) {
			output += "All Organizations are Ready for User Migrations!\n\n"
		}

		if len(notInPsa) > 0 {
			output += lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				Bold(true).
				Render("Organizations Not in PSA")
			output += fmt.Sprintf("\n%s\n\n", strings.Join(notInPsa, "\n"))
		}

		if len(notReady) > 0 {
			output += lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				Bold(true).
				Render("Organizations Not Ready for User Migration")
			output += fmt.Sprintf("\n%s\n\n", strings.Join(notReady, "\n"))
		}

		if len(ready) > 0 {
			output += lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				Bold(true).
				Render("Organizations Ready for User Migration")
			output += fmt.Sprintf("\n%s\n\n", strings.Join(ready, "\n"))
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
