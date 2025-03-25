package tui

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"sort"
	"strconv"
	"strings"
)

var testLimit = 5

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

type ticketMigrationDetails struct {
	ZendeskTicket *zendesk.Ticket `json:"zendesk_ticket"`
	PsaTicket     *psa.Ticket     `json:"psa_ticket"`

	Migrated bool `json:"migrated"`
}

type ticketMigTotals struct {
	totalOrgsToMigrateTickets int
	totalOrgsChecked          int
	totalTicketsToMigrate     int
	totalAlreadyMigrated      int
	totalTicketsMigrated      int
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

type fatalErrMsg struct {
	Msg string
	Err error
}

const (
	ticketMigNoOrgs            ticketMigStatus = "noOrgs"
	ticketMigWaitingForOrgs    ticketMigStatus = "waitingForOrgs"
	ticketMigPickingOrgs       ticketMigStatus = "pickingOrgs"
	ticketMigGettingPsaTickets ticketMigStatus = "gettingPsaTickets"
	ticketMigMigratingTickets  ticketMigStatus = "migratingTickets"
	ticketMigDone              ticketMigStatus = "ticketMigDone"
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
		m.status = ticketMigStatus(msg)
		switch msg {

		case switchTicketMigStatusMsg(ticketMigPickingOrgs):
			slog.Debug("got pickingOrgs status")
			m.form = m.orgSelectionForm()

		case switchTicketMigStatusMsg(ticketMigGettingPsaTickets):
			return m, tea.Sequence(m.getAlreadyMigrated(), saveDataCmd())

		case switchTicketMigStatusMsg(ticketMigMigratingTickets):
			for _, org := range m.data.Orgs {
				if org.OrgMigrated && org.TicketMigSelected {
					cmds = append(cmds, m.runMigration(org))
				}
			}

			slog.Debug("ticket migration: converting tickets", "totalOrgs", m.totalOrgsToMigrateTickets, "totalTickets", m.totalTicketsToMigrate)
			return m, tea.Sequence(cmds...)
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
					m.totalOrgsToMigrateTickets++
				}

			} else {
				for _, org := range m.selectedOrgs {
					org.TicketMigSelected = true
					m.totalOrgsToMigrateTickets++
				}
			}

			cmds = append(cmds, switchTicketMigStatus(ticketMigGettingPsaTickets))
		}
	}

	if m.status == ticketMigMigratingTickets && m.totalOrgsToMigrateTickets == m.totalOrgsDone {
		cmds = append(cmds, switchTicketMigStatus(ticketMigDone), saveDataCmd())
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
	case ticketMigGettingPsaTickets:
		s = runSpinner(fmt.Sprintf("Getting existing PSA tickets...%d found", m.totalAlreadyMigrated))
	case ticketMigMigratingTickets:
		s = runSpinner(fmt.Sprintf("Migrating tickets...total done: %d/%d", m.totalTicketsMigrated, m.totalTicketsToMigrate))
	case ticketMigDone:
		s = fmt.Sprintf("Ticket migration done - press %s to run again\n\n", textNormalAdaptive("SPACE"))
	}

	return s
}

func (m *ticketMigrationModel) runMigration(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		zTickets, err := m.getZendeskTickets(org)
		if err != nil {
			m.totalErrors++
			slog.Error("runMigration: getting tickets for org", "orgName", org.ZendeskOrg.Name, "error", err)
			m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("getting zendesk tickets for org %s: %w", org.ZendeskOrg.Name, err)))
			return nil
		}

		var ticketsToMigrate []*ticketMigrationDetails
		for _, ticket := range zTickets {
			if testLimit > 0 && m.totalTicketsMigrated >= testLimit {
				slog.Info("testLimit reached")
				m.totalOrgsDone++
				break
			}

			if psaId, ok := m.data.TicketsInPsa[strconv.Itoa(ticket.Id)]; !ok {
				td := &ticketMigrationDetails{
					ZendeskTicket: &ticket,
					PsaTicket:     &psa.Ticket{},
				}

				slog.Debug("ticket needs to be migrated", "zendeskId", ticket.Id)
				ticketsToMigrate = append(ticketsToMigrate, td)
				m.totalTicketsToMigrate++

			} else {
				slog.Debug("ticket already migrated", "zendeskId", ticket.Id, "psaId", psaId)
			}
		}

		for _, ticket := range ticketsToMigrate {
			var err error
			ticket.PsaTicket, err = m.createBaseTicket(org, ticket)
			if err != nil {
				slog.Error("error creating base ticket", "orgName", org.ZendeskOrg.Name, "ticketId", ticket.ZendeskTicket.Id, "error", err)
				m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("creating base ticket %s ticket %d to psa ticket: %w", org.ZendeskOrg.Name, ticket.ZendeskTicket.Id, err)))
				m.totalErrors++
				continue
			}

			comments, err := m.client.ZendeskClient.GetAllTicketComments(ctx, int64(ticket.ZendeskTicket.Id))
			if err != nil {
				slog.Error("getting comments for zendesk ticket", "orgName", org.ZendeskOrg.Name, "ticketId", ticket.ZendeskTicket.Id, "error", err)
				m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("getting comments for %s ticket %d: %w", org.ZendeskOrg.Name, ticket.ZendeskTicket.Id, err)))
				m.totalErrors++
				continue
			}

			if err := m.createTicketNotes(org, ticket, comments); err != nil {
				slog.Error("creating comments for connectwise ticket", "orgName", org.ZendeskOrg.Name, "ticketId", ticket.PsaTicket.Id, "error", err)
				m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("creating comments on ticket %d: %w", ticket.PsaTicket.Id, err)))
				m.totalErrors++
				continue
			}

			if ticket.ZendeskTicket.Status == "closed" {
				if err := m.client.CwClient.UpdateTicketStatus(ctx, ticket.PsaTicket, m.data.PsaInfo.StatusClosed.Id); err != nil {
					slog.Error("error closing ticket", "orgName", org.ZendeskOrg.Name, "zendeskTicketId", ticket.ZendeskTicket.Id, "psaTicketId", ticket.PsaTicket.Id, "error", err)
					m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("closing %s psa ticket %d: %w", org.ZendeskOrg.Name, ticket.PsaTicket.Id, err)))
					m.totalErrors++
					continue
				}
			}

			m.data.writeToOutput(goodGreenOutput("CREATED", fmt.Sprintf("migrated ticket: %d", ticket.ZendeskTicket.Id)))
			m.totalTicketsMigrated++
		}

		m.totalOrgsDone++
		return nil
	}
}

func (m *ticketMigrationModel) getAlreadyMigrated() tea.Cmd {
	return func() tea.Msg {
		s := fmt.Sprintf("id=%d AND value != null", m.data.PsaInfo.ZendeskTicketFieldId.Id)
		tickets, err := m.client.CwClient.GetTickets(ctx, &s)
		if err != nil {
			return fatalErrMsg{
				Msg: "getting existing tickets from psa",
				Err: err,
			}
		}

		for _, ticket := range tickets {
			for _, field := range ticket.CustomFields {
				if field.Id == m.data.PsaInfo.ZendeskTicketFieldId.Id {
					// if value is an int, it's a zendesk ticket id
					if _, ok := field.Value.(float64); ok {
						val := strconv.Itoa(int(field.Value.(float64)))
						m.data.TicketsInPsa[val] = ticket.Id
						m.totalAlreadyMigrated++
						break
					}
				}
			}
		}

		return switchTicketMigStatusMsg(ticketMigMigratingTickets)
	}
}

func (m *ticketMigrationModel) getZendeskTickets(org *orgMigrationDetails) ([]zendesk.Ticket, error) {
	slog.Info("getting tickets for org", "orgName", org.ZendeskOrg.Name)
	q := zendesk.SearchQuery{
		TicketsOrganizationId: org.ZendeskOrg.Id,
		TicketCreatedAfter:    org.Tag.StartDate,
		TicketCreatedBefore:   org.Tag.EndDate,
	}

	tickets, err := m.client.ZendeskClient.GetTicketsWithQuery(ctx, q, 100, testLimit)
	if err != nil {
		slog.Error("getting tickets for org", "orgName", org.ZendeskOrg.Name, "error", err)
		return nil, fmt.Errorf("getting tickets via zendesk api: %w", err)
	}
	m.data.writeToOutput(infoOutput("INFO", fmt.Sprintf("got ticket total for %s: %d", org.ZendeskOrg.Name, len(tickets))))
	return tickets, nil
}

func (m *ticketMigrationModel) createBaseTicket(org *orgMigrationDetails, ticket *ticketMigrationDetails) (*psa.Ticket, error) {
	if ticket.ZendeskTicket == nil {
		return nil, errors.New("zendesk ticket does not exist")
	}

	customField := *m.data.PsaInfo.ZendeskTicketFieldId
	customField.Value = ticket.ZendeskTicket.Id

	baseTicket := &psa.Ticket{
		Board:        m.data.PsaInfo.Board,
		Status:       m.data.PsaInfo.StatusOpen,
		Summary:      ticket.ZendeskTicket.Subject,
		Company:      &psa.Company{Id: org.PsaOrg.Id},
		CustomFields: []psa.CustomField{customField},
	}

	baseTicket.Summary = ticket.ZendeskTicket.Subject
	if len(baseTicket.Summary) > 100 {
		baseTicket.Summary = baseTicket.Summary[:100]
		baseTicket.InitialInternalAnalysis = fmt.Sprintf("Ticket subject was shortened by migration utility (maximum ticket summary in ConnectWise PSA is 100 characters)\n\n"+
			"Original Subject: %s", ticket.ZendeskTicket.Subject)
	}

	userString := strconv.Itoa(int(ticket.ZendeskTicket.RequesterId))
	if user, ok := org.UsersToMigrate[userString]; ok {
		baseTicket.Contact = &psa.Contact{Id: user.PsaContact.Id}
	} else {
		return nil, fmt.Errorf("couldn't find user for ticket requester: %s", userString)
	}

	ownerString := strconv.Itoa(int(ticket.ZendeskTicket.AssigneeId))
	if owner, ok := m.client.Cfg.AgentMappings[ownerString]; ok {
		if owner.PsaId != 0 {
			baseTicket.Owner = &psa.Owner{Id: owner.PsaId}
		}
	}

	var err error
	baseTicket, err = m.client.CwClient.PostTicket(ctx, baseTicket)
	if err != nil {
		return nil, fmt.Errorf("posting base ticket to connectwise: %w", err)
	}

	return baseTicket, nil
}

func (m *ticketMigrationModel) createTicketNotes(org *orgMigrationDetails, ticket *ticketMigrationDetails, comments []zendesk.Comment) error {
	for _, comment := range comments {
		note := &psa.TicketNote{}

		authorString := strconv.Itoa(int(comment.AuthorId))
		slog.Debug("author id", "authorId", authorString)
		if agent, ok := m.client.Cfg.AgentMappings[authorString]; ok {
			slog.Debug("author is in agent mappings", "authorId", authorString)
			note.Member = &psa.Member{Id: agent.PsaId}
		} else if contact, ok := org.UsersToMigrate[authorString]; ok {
			slog.Debug("author is in org data", "authorId", authorString)
			note.Contact = &psa.Contact{Id: contact.PsaContact.Id}
		} else {
			senderName := comment.Via.Source.From.Name
			senderEmail := "no email"
			if comment.Via.Source.From.Address != "" {
				senderEmail = comment.Via.Source.From.Address
			}

			note.Text += fmt.Sprintf("Sent By: %s (%s)\n", senderName, senderEmail)

		}

		if comment.Public {
			note.DetailDescriptionFlag = true
		} else {
			note.InternalAnalysisFlag = true
		}

		note.Text += fmt.Sprintf("%s\n", comment.CreatedAt.Format("1/2/2006 3:04PM"))

		ccs := m.getCcString(org, &comment)
		if ccs != "" {
			note.Text += fmt.Sprintf("CCs: %s\n", ccs)
		}

		note.Text += fmt.Sprintf("\n%s", comment.PlainBody)

		if err := m.client.CwClient.PostTicketNote(ctx, ticket.PsaTicket.Id, note); err != nil {
			return fmt.Errorf("creating note in ticket: %w", err)
		}
	}

	return nil
}

func (m *ticketMigrationModel) getCcString(org *orgMigrationDetails, comment *zendesk.Comment) string {
	var ccs []string
	for _, cc := range comment.Via.Source.To.EmailCcs {
		// check if cc is a string
		if cc, ok := cc.(string); ok {
			ccs = append(ccs, cc)
			continue
		}

		if cc, ok := cc.(int); ok {
			ccString := strconv.Itoa(cc)
			if agent, ok := m.client.Cfg.AgentMappings[ccString]; ok {
				slog.Debug("cc is in zendesk agent mappings", "agent", ccString, "email", agent.Email)
				ccs = append(ccs, agent.Email)
			} else {
				if contact, ok := org.UsersToMigrate[ccString]; ok {
					slog.Debug("cc is in org data", "id", ccString, "email", contact.ZendeskUser.Email)
					ccs = append(ccs, contact.ZendeskUser.Email)
				}
			}
		}
	}
	return strings.Join(ccs, ", ")
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
