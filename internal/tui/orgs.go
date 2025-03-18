package tui

import (
	"errors"
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

type orgMigrationModel struct {
	client       *migration.Client
	data         *MigrationData
	tags         []tagDetails
	checkedTotal int
	//orgs   *allOrgs
	status orgMigStatus
	done   bool
}

type tagDetails struct {
	name      string
	startDate time.Time
	endDate   time.Time
}

//type allOrgs struct {
//	master      []*orgMigrationDetails
//	checked     []*orgMigrationDetails
//	toMigrate   []*orgMigrationDetails
//	erroredOrgs []erroredOrg
//}

type erroredOrg struct {
	org *orgMigrationDetails
	err error
}

type orgMigStatus string

type switchOrgMigStatusMsg string

func switchOrgMigStatus(s orgMigStatus) tea.Cmd {
	return func() tea.Msg {
		return switchOrgMigStatusMsg(s)
	}
}

const (
	awaitingStart      orgMigStatus = "awaitingStart"
	gettingTags        orgMigStatus = "gettingTags"
	gettingZendeskOrgs orgMigStatus = "gettingZendeskOrgs"
	comparingOrgs      orgMigStatus = "comparingOrgs"
	orgMigDone         orgMigStatus = "orgMigDone"
)

func newOrgMigrationModel(mc *migration.Client, data *MigrationData) *orgMigrationModel {
	return &orgMigrationModel{
		client: mc,
		data:   data,
		status: awaitingStart,
	}
}

func (m *orgMigrationModel) Init() tea.Cmd {
	return nil
}

func (m *orgMigrationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case " ":
			if m.status == awaitingStart {
				slog.Debug("org checker: user pressed space to start")
				return m, switchOrgMigStatus(gettingTags)
			}
		}

	case switchOrgMigStatusMsg:
		switch msg {
		case switchOrgMigStatusMsg(gettingTags):

			m.status = gettingTags
			return m, m.getTagDetails()

		case switchOrgMigStatusMsg(gettingZendeskOrgs):
			slog.Debug("org checker: got tags", "tags", m.tags)
			m.status = gettingZendeskOrgs
			return m, m.getOrgs()

		case switchOrgMigStatusMsg(comparingOrgs):
			slog.Debug("org checker: got orgs", "total", len(m.data.Orgs))
			m.status = comparingOrgs
			var checkOrgCmds []tea.Cmd
			for _, org := range m.data.Orgs {
				checkOrgCmds = append(checkOrgCmds, m.checkOrg(org))
			}
			return m, tea.Batch(checkOrgCmds...)

		case switchOrgMigStatusMsg(orgMigDone):
			slog.Debug("org checker: done checking orgs")
			m.status = orgMigDone
			cmds = append(cmds, m.constructOutput())
			cmds = append(cmds, sendOrgsCmd(m.data.Orgs))
		}
	}

	if m.status == comparingOrgs && !m.done {
		if len(m.data.Orgs) == m.checkedTotal {
			cmd = switchOrgMigStatus(orgMigDone)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *orgMigrationModel) View() string {
	var s, st string
	switch m.status {
	case awaitingStart:
		st = "Press SPACE to begin org migration"
	case gettingTags:
		st = runSpinner("Getting Zendesk tags")
	case gettingZendeskOrgs:
		st = runSpinner("Getting Zendesk orgs")
	case comparingOrgs:
		st = runSpinner("Checking for PSA orgs")
	case orgMigDone:
		st = "Done - press 'm' to return to the main menu"
	}

	s += st

	if m.status != awaitingStart {
		s += fmt.Sprintf("\n\nChecked: %d/%d\n", m.checkedTotal, len(m.data.Orgs))
	}

	return s
}

func (m *orgMigrationModel) getTagDetails() tea.Cmd {
	return func() tea.Msg {
		for _, tag := range m.client.Cfg.Zendesk.TagsToMigrate {
			tm := &timeConversionDetails{
				startString:   tag.StartDate,
				endString:     tag.EndDate,
				startFallback: m.client.Cfg.Zendesk.MasterStartDate,
				endFallback:   m.client.Cfg.Zendesk.MasterEndDate,
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
		return switchOrgMigStatusMsg(gettingZendeskOrgs)
	}
}

func (m *orgMigrationModel) getOrgs() tea.Cmd {
	return func() tea.Msg {
		slog.Info("getting orgs for tags", "tags", m.client.Cfg.Zendesk.TagsToMigrate)
		for _, tag := range m.tags {
			slog.Debug("getting orgs for tag", "tag", tag.name)
			q := &zendesk.SearchQuery{}
			q.Tags = []string{tag.name}

			slog.Info("getting all orgs from zendesk for tag group", "tag", tag.name)

			orgs, err := m.client.ZendeskClient.GetOrganizationsWithQuery(ctx, *q)
			if err != nil {
				return apiErrMsg{err}
			}

			for _, org := range orgs {
				md := &orgMigrationDetails{
					ZendeskOrg: &org,
					Tag:        &tag,
				}
				m.data.Orgs = append(m.data.Orgs, md)
			}
		}
		return switchOrgMigStatusMsg(comparingOrgs)
	}
}

func (m *orgMigrationModel) checkOrg(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		q := &zendesk.SearchQuery{}
		if org.Tag.startDate != (time.Time{}) {
			q.TicketCreatedAfter = org.Tag.startDate
		}

		if org.Tag.endDate != (time.Time{}) {
			q.TicketCreatedBefore = org.Tag.endDate
		}

		q.TicketsOrganizationId = org.ZendeskOrg.Id

		tickets, err := m.client.ZendeskClient.GetTicketsWithQuery(ctx, *q, 20, true)
		if err != nil {
			slog.Error("getting tickets for org", "orgName", org.ZendeskOrg.Name, "error", err)
			org.OrgMigErrors = append(org.OrgMigErrors, err)
			m.checkedTotal++
			return nil
		}

		if len(tickets) > 0 {
			org.HasTickets = true
		} else {
			// We only care about orgs with tickets - no need to check further
			m.checkedTotal++
			return nil
		}

		org.PsaOrg, err = m.matchZdOrgToCwCompany(org.ZendeskOrg)
		if err != nil {
			slog.Warn("org is not in PSA", "orgName", org.ZendeskOrg.Name)
			m.checkedTotal++
			return nil
		}

		if err := m.updateCompanyFieldValue(org); err != nil {
			slog.Error("updating company field value in zendesk", "error", err)
			org.OrgMigErrors = append(org.OrgMigErrors, err)
			m.checkedTotal++
			return nil
		}

		if org.PsaOrg != nil && org.ZendeskOrg.OrganizationFields.PSACompanyId == int64(org.PsaOrg.Id) {
			org.ReadyUsers = true
			slog.Debug("org is ready for user migration", "orgName", org.ZendeskOrg.Name)
		}

		m.checkedTotal++
		return nil
	}
}

func (m *orgMigrationModel) updateCompanyFieldValue(org *orgMigrationDetails) error {
	if org.ZendeskOrg.OrganizationFields.PSACompanyId == int64(org.PsaOrg.Id) {
		slog.Debug("zendesk org already has PSA company id field", "orgName", org.ZendeskOrg.Name, "psaCompanyId", org.ZendeskOrg.OrganizationFields.PSACompanyId)
		return nil
	}

	if org.PsaOrg.Id != 0 {
		org.ZendeskOrg.OrganizationFields.PSACompanyId = int64(org.PsaOrg.Id)

		var err error
		org.ZendeskOrg, err = m.client.ZendeskClient.UpdateOrganization(ctx, org.ZendeskOrg)
		if err != nil {
			return fmt.Errorf("updating organization with PSA company id: %w", err)
		}

		slog.Info("updated zendesk organization with PSA company id", "orgName", org.ZendeskOrg.Name, "psaCompanyId", org.PsaOrg.Id)
		return nil
	} else {
		slog.Error("org psa id is 0 - cannot update psa_company field in zendesk", "orgName", org.ZendeskOrg.Name)
		return errors.New("org psa id is 0 - cannot update psa_company field in zendesk")
	}
}

func (m *orgMigrationModel) matchZdOrgToCwCompany(org *zendesk.Organization) (*psa.Company, error) {
	comp, err := m.client.CwClient.GetCompanyByName(ctx, org.Name)
	if err != nil {
		return nil, err
	}

	return comp, nil
}

func (m *orgMigrationModel) constructOutput() tea.Cmd {
	return func() tea.Msg {
		var tagNames, withTickets, notInPsa, notReady, readyUsers, errors []string

		for _, tag := range m.tags {
			tagNames = append(tagNames, tag.name)
		}

		for _, org := range m.data.Orgs {
			if org.HasTickets {
				withTickets = append(withTickets, org.ZendeskOrg.Name)
			}

			if org.PsaOrg == nil {
				notInPsa = append(notInPsa, org.ZendeskOrg.Name)
			}

			if org.ReadyUsers {
				readyUsers = append(readyUsers, org.ZendeskOrg.Name)
			} else {
				notReady = append(notReady, org.ZendeskOrg.Name)
			}

			if len(org.OrgMigErrors) > 0 {
				for _, e := range org.OrgMigErrors {
					errors = append(errors, fmt.Sprintf("%s: %s", org.ZendeskOrg.Name, e))
				}
			}
		}

		slices.Sort(tagNames)
		slices.Sort(withTickets)
		slices.Sort(notInPsa)
		slices.Sort(notReady)
		slices.Sort(readyUsers)
		slices.Sort(errors)

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
			len(readyUsers), len(withTickets),
			len(errors))

		if len(withTickets) == len(readyUsers) {
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

		if len(readyUsers) > 0 {
			output += lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				Bold(true).
				Render("Organizations Ready for User Migration")
			output += fmt.Sprintf("\n%s\n\n", strings.Join(readyUsers, "\n"))
		}

		if len(errors) > 0 {
			output += lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				Bold(true).
				Render("Errors")
			output += fmt.Sprintf("\n%s\n", strings.Join(errors, "\n"))
		}

		return updateResultsMsg{body: output}
	}
}
