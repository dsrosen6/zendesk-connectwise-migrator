package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"sort"
)

type ticketMigrationModel struct {
	client          *migration.Client
	data            *MigrationData
	form            *huh.Form
	selectedOrgs    []*orgMigrationDetails
	allOrgsSelected bool
	ticketMigTotals

	status ticketMigStatus
	done   bool
}

type ticketMigTotals struct {
	totalOrgsToMigrateTickets int
	totalOrgsChecked          int
	totalTicketsToProcess     int
	totalBaseTicketsCreated   int
	totalCommentsCreated      int
	totalOrgsDone             int
	totalErrors               int
}

type ticketMigStatus string

type switchTicketMigStatusMsg string

func switchTicketMigStatus(s ticketMigStatus) tea.Cmd {
	return func() tea.Msg {
		return switchTicketMigStatusMsg(s)
	}
}

const (
	ticketMigNoOrgs         ticketMigStatus = "noOrgs"
	ticketMigWaitingForOrgs ticketMigStatus = "waitingForOrgs"
	ticketMigPickingOrgs    ticketMigStatus = "pickingOrgs"
	ticketMigGettingTickets ticketMigStatus = "gettingTickets"
	ticketMigDone           ticketMigStatus = "ticketMigDone"
)

type ticketMigInitFormMsg struct{}

func ticketMigInitForm() tea.Cmd {
	return func() tea.Msg {
		return ticketMigInitFormMsg{}
	}
}

func newTicketMigrationModel(mc *migration.Client, data *MigrationData) *ticketMigrationModel {
	return &ticketMigrationModel{
		client: mc,
		data:   data,
		status: ticketMigNoOrgs,
	}
}

func (m *ticketMigrationModel) Init() tea.Cmd {
	return nil
}

func (m *ticketMigrationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ticketMigInitFormMsg:
		if len(m.data.Orgs) == 0 {
			slog.Warn("got ticketMigInitFormMsg, but no orgs")
			m.data.writeToOutput(warnYellowOutput("WARNING", "ticket migration - no orgs"))
			return m, switchTicketMigStatus(ticketMigNoOrgs)
		} else {
			slog.Debug("got ticketMigInitFormMsg", "totalOrgs", len(m.data.Orgs))
			m.form = m.orgSelectionForm()
			cmds = append(cmds, m.form.Init(), switchTicketMigStatus(ticketMigPickingOrgs))
			return m, tea.Sequence(cmds...)
		}

	case switchTicketMigStatusMsg:
		switch msg {
		case switchTicketMigStatusMsg(ticketMigNoOrgs):
			m.status = ticketMigNoOrgs

		case switchTicketMigStatusMsg(ticketMigWaitingForOrgs):
			m.status = ticketMigWaitingForOrgs

		case switchTicketMigStatusMsg(ticketMigPickingOrgs):
			slog.Debug("got pickingOrgs status")
			m.form = m.orgSelectionForm()
			m.status = ticketMigPickingOrgs

		case switchTicketMigStatusMsg(ticketMigGettingTickets):
			m.ticketMigTotals = ticketMigTotals{}
			m.status = ticketMigGettingTickets
			for _, org := range m.data.Orgs {
				if org.OrgMigrated && org.TicketMigSelected {
					m.totalOrgsToMigrateTickets++
					cmds = append(cmds, m.getTicketsToMigrate(org))
				}
			}

			slog.Debug("ticket migration: orgs picked", "totalOrgs", m.totalOrgsToMigrateTickets)
			return m, tea.Sequence(cmds...)

		case switchTicketMigStatusMsg(ticketMigDone):
			m.status = ticketMigDone
			cmds = append(cmds, saveDataCmd())
		}
	}

	if m.status == ticketMigPickingOrgs {
		form, cmd := m.form.Update(msg)
		cmds = append(cmds, cmd)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		if m.form.State == huh.StateCompleted {

			if m.allOrgsSelected {
				for _, org := range m.data.Orgs {
					org.TicketMigSelected = true
				}

			} else {
				for _, org := range m.selectedOrgs {
					org.TicketMigSelected = true
				}
			}

			cmds = append(cmds, switchTicketMigStatus(ticketMigGettingTickets))
		}
	}

	if m.status == ticketMigGettingTickets && m.totalOrgsToMigrateTickets == m.totalOrgsChecked {
		cmds = append(cmds, switchTicketMigStatus(ticketMigDone))
	}

	return m, tea.Batch(cmds...)
}

func (m *ticketMigrationModel) View() string {
	var s string
	switch m.status {
	case ticketMigNoOrgs:
		s = "No orgs have been loaded! Please return to the main menu and select Organizations, then return."
	case ticketMigWaitingForOrgs:
		s = runSpinner("Org migration is running - please wait")
	case ticketMigPickingOrgs:
		s = m.form.View()
	case ticketMigGettingTickets:
		s = runSpinner("Getting tickets")
	case ticketMigDone:
		s = fmt.Sprintf("User migration done - press %s to run again\n\n", textNormalAdaptive("SPACE"))
	}

	return s
}

func (m *ticketMigrationModel) getTicketsToMigrate(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		slog.Info("getting tickets for org", "orgName", org.ZendeskOrg.Name)
		q := zendesk.SearchQuery{
			TicketsOrganizationId: org.ZendeskOrg.Id,
			TicketCreatedAfter:    org.Tag.StartDate,
			TicketCreatedBefore:   org.Tag.EndDate,
		}

		tickets, err := m.client.ZendeskClient.GetTicketsWithQuery(ctx, q, 100, false)
		if err != nil {
			slog.Error("getting tickets for org", "orgName", org.ZendeskOrg.Name, "error", err)
			m.totalOrgsChecked++
			m.totalErrors++
			return nil
		}
		m.data.writeToOutput(infoOutput("INFO", fmt.Sprintf("got ticket total for %s: %d", org.ZendeskOrg.Name, len(tickets))))

		for _, ticket := range tickets {
			idString := fmt.Sprintf("%d", ticket.Id)
			if _, ok := org.TicketsToMigrate[idString]; !ok {
				tm := &ticketMigrationDetails{
					ZendeskTicket: &ticket,
				}
				org.addTicketToOrgMap(idString, tm)
				m.totalTicketsToProcess++
			}
		}
		m.totalOrgsChecked++
		return nil
	}
}

func (md *orgMigrationDetails) addTicketToOrgMap(idString string, ticket *ticketMigrationDetails) {
	md.TicketsToMigrate[idString] = ticket
}

func (m *ticketMigrationModel) orgSelectionForm() *huh.Form {
	return huh.NewForm(

		huh.NewGroup(
			huh.NewSelect[bool]().
				Title("Migrate all confirmed orgs?").
				Description("If not, select the organizations you want to migrate on the next screen.").
				Options(
					huh.NewOption("All Orgs", true),
					huh.NewOption("Select Orgs", false)).
				Value(&m.allOrgsSelected),
		),
		huh.NewGroup(
			huh.NewMultiSelect[*orgMigrationDetails]().
				Title("Pick the orgs you'd like to migrate users for").
				Description("Use Space to select, and Enter/Return to submit").
				Options(m.orgOptions()...).
				Value(&m.selectedOrgs),
		).WithHideFunc(func() bool { return m.allOrgsSelected == true }),
	).WithHeight(verticalLeftForMainView).WithShowHelp(false).WithTheme(migration.CustomHuhTheme())
}

func (m *ticketMigrationModel) orgOptions() []huh.Option[*orgMigrationDetails] {
	var orgOptions []huh.Option[*orgMigrationDetails]
	for _, org := range m.data.Orgs {
		if org.OrgMigrated {
			opt := huh.Option[*orgMigrationDetails]{
				Key:   org.ZendeskOrg.Name,
				Value: org,
			}

			orgOptions = append(orgOptions, opt)
		}
	}

	sort.Slice(orgOptions, func(i, j int) bool {
		return orgOptions[i].Key < orgOptions[j].Key
	})

	return orgOptions
}
