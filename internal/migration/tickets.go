package migration

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"strconv"
	"strings"
	"sync"
)

const (
	totalConcurrentTickets = 25
)

type ticketMigrationDetails struct {
	ZendeskTicket *zendesk.Ticket
	PsaTicket     *psa.Ticket

	Migrated bool
}

type activeTicketMigration struct {
	orgName          string
	status           ticketStatus
	ticketsToProcess int
	ticketsProcessed int
}

func (m *Model) runTicketMigration(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		slog.Debug("runTicketMigration: called", "orgName", org.ZendeskOrg.Name)
		m.currentTicketMigration = newActiveTicketMigration(org.ZendeskOrg.Name, ticketStatusGetting)

		zTickets, err := m.getZendeskTickets(org)
		if err != nil {
			m.ticketMigrationErrors++
			slog.Error("getting tickets for org", "orgName", org.ZendeskOrg.Name, "error", err)
			m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("%s: couldn't get zendesk tickets: %s", org.ZendeskOrg.Name, err)), errOutput)
			return nil
		}

		m.currentTicketMigration.ticketsProcessed = org.TicketsAlreadyInPSA
		m.currentTicketMigration.ticketsToProcess = len(zTickets)
		m.currentTicketMigration.status = ticketStatusMigrating

		var ticketsToMigrate []*ticketMigrationDetails
		for _, ticket := range zTickets {
			if psaId, ok := m.data.TicketsInPsa[strconv.Itoa(ticket.Id)]; !ok {
				td := &ticketMigrationDetails{
					ZendeskTicket: &ticket,
					PsaTicket:     &psa.Ticket{},
				}

				slog.Debug("runTicketMigration: ticket needs to be migrated", "zendeskId", ticket.Id)
				ticketsToMigrate = append(ticketsToMigrate, td)

			} else {
				slog.Debug("runTicketMigration: ticket already migrated", "zendeskId", ticket.Id, "psaId", psaId)
			}
		}

		sem := make(chan struct{}, totalConcurrentTickets)
		var wg sync.WaitGroup

		for _, ticket := range ticketsToMigrate {
			sem <- struct{}{}
			wg.Add(1)

			go func(ticket *ticketMigrationDetails) {
				defer wg.Done()
				defer func() { <-sem }()

				if m.client.Cfg.TicketLimit > 0 && m.ticketsProcessed >= m.client.Cfg.TicketLimit {
					slog.Info("testLimit reached")
					m.ticketOrgsProcessed++
					return
				}

				slog.Debug("creating base ticket", "zendeskId", ticket.ZendeskTicket.Id)
				var err error
				ticket.PsaTicket, err = m.createBaseTicket(org, ticket)
				if err != nil {
					var noUserErr NoUserErr
					if errors.As(err, &noUserErr) {
						m.ticketMigrationErrors++
						m.ticketsProcessed++
						m.currentTicketMigration.ticketsProcessed++
						slog.Warn("creating base ticket: no user found", "orgName", org.ZendeskOrg.Name, "zendeskTicketId", ticket.ZendeskTicket.Id, "userId", noUserErr.UserId)
						m.writeToOutput(warnYellowOutput("WARN", fmt.Sprintf("%s: couldn't convert ticket %d to psa ticket: no user found", org.ZendeskOrg.Name, ticket.ZendeskTicket.Id)), errOutput)
						return
					}

					slog.Error("creating base ticket", "orgName", org.ZendeskOrg.Name, "zendeskTicketId", ticket.ZendeskTicket.Id, "error", err)
					m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("%s: couldn't convert ticket %d to psa ticket: %s", org.ZendeskOrg.Name, ticket.ZendeskTicket.Id, err)), errOutput)
					m.ticketMigrationErrors++
					m.ticketsProcessed++
					m.currentTicketMigration.ticketsProcessed++
					return
				}

				comments, err := m.client.ZendeskClient.GetAllTicketComments(m.ctx, int64(ticket.ZendeskTicket.Id))
				if err != nil {
					slog.Error("getting comments for zendesk ticket", "orgName", org.ZendeskOrg.Name, "zendeskTicketId", ticket.ZendeskTicket.Id, "error", err)
					m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("%s: error getting comments for ticket %d", org.ZendeskOrg.Name, ticket.ZendeskTicket.Id)), errOutput)
					m.ticketMigrationErrors++
					m.ticketsProcessed++
					m.currentTicketMigration.ticketsProcessed++
					return
				}

				slog.Debug("creating ticket notes", "zendeskId", ticket.ZendeskTicket.Id, "psaId", ticket.PsaTicket.Id)
				if err := m.createTicketNotes(ticket, comments); err != nil {
					slog.Error("creating comments for connectwise ticket", "orgName", org.ZendeskOrg.Name, "zendeskTicketId", ticket.ZendeskTicket.Id, "psaTicketId", ticket.PsaTicket.Id, "error", err)
					m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("%s: couldn't create comments for ticket %d: %s", org.ZendeskOrg.Name, ticket.PsaTicket.Id, err)), errOutput)
					m.ticketMigrationErrors++
					m.ticketsProcessed++
					m.currentTicketMigration.ticketsProcessed++
					return
				}

				if ticket.ZendeskTicket.Status == "closed" || ticket.ZendeskTicket.Status == "solved" {
					slog.Debug("runTicketMigration: closing ticket", "closedOn", ticket.ZendeskTicket.UpdatedAt)
					if err := m.client.CwClient.UpdateTicketStatus(m.ctx, ticket.PsaTicket, m.data.PsaInfo.StatusClosed.Id); err != nil {
						slog.Error("closing ticket", "orgName", org.ZendeskOrg.Name, "zendeskTicketId", ticket.ZendeskTicket.Id, "psaTicketId", ticket.PsaTicket.Id, "error", err)
						m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("%s: couldn't close psa ticket %d: %s", org.ZendeskOrg.Name, ticket.PsaTicket.Id, err)), errOutput)
						m.ticketMigrationErrors++
						m.ticketsProcessed++
						m.currentTicketMigration.ticketsProcessed++
						return
					}
				}

				slog.Debug("runTicketMigration: migration complete for ticket", "orgName", org.ZendeskOrg.Name, "zendeskTicketId", ticket.ZendeskTicket.Id, "psaTicketId", ticket.PsaTicket.Id)
				m.mu.Lock()
				m.data.TicketsInPsa[strconv.Itoa(ticket.ZendeskTicket.Id)] = ticket.PsaTicket.Id
				m.mu.Unlock()
				m.newTicketsCreated++
				m.ticketsProcessed++
				m.currentTicketMigration.ticketsProcessed++
			}(ticket)
		}

		wg.Wait()
		m.ticketOrgsProcessed++
		return nil
	}
}

func (m *Model) getAlreadyMigrated() tea.Cmd {
	return func() tea.Msg {
		s := fmt.Sprintf("id=%d AND value != null", m.data.PsaInfo.ZendeskTicketIdField.Id)
		tickets, err := m.client.CwClient.GetTickets(m.ctx, &s)
		if err != nil {
			m.writeToOutput(badRedOutput("FATAL ERROR", fmt.Sprintf("getting already migrated tickets: %s", err)), errOutput)
			return fatalErrMsg{
				Msg: "getting existing tickets from psa",
				Err: err,
			}
		}

		slog.Debug("getAlreadyMigrated: found already migrated tickets", "count", len(tickets))
		for _, ticket := range tickets {
			for _, field := range ticket.CustomFields {
				if field.Id == m.data.PsaInfo.ZendeskTicketIdField.Id {
					// if value is an int, it's a zendesk ticket id
					if _, ok := field.Value.(float64); ok {
						val := strconv.Itoa(int(field.Value.(float64)))
						m.data.TicketsInPsa[val] = ticket.Id
						m.ticketsProcessed++
						for _, org := range m.data.SelectedOrgs {
							if ticket.Company.Id == org.PsaOrg.Id {
								org.TicketsAlreadyInPSA++
								break
							}
						}
						break
					}
				}
			}
		}

		return switchStatusMsg(migratingTickets)
	}
}

func (m *Model) getZendeskTickets(org *orgMigrationDetails) ([]zendesk.Ticket, error) {
	slog.Debug("getZendeskTickets: called", "orgName", org.ZendeskOrg.Name)
	q := zendesk.SearchQuery{
		TicketsOrganizationId: org.ZendeskOrg.Id,
		TicketCreatedAfter:    org.Tag.StartDate,
		TicketCreatedBefore:   org.Tag.EndDate,
	}

	if m.client.Cfg.MigrateOpenTickets {
		q.GetOpenTickets = true
	}

	tickets, err := m.client.ZendeskClient.GetTicketsWithQuery(m.ctx, q, 100, m.client.Cfg.TicketLimit)
	if err != nil {
		slog.Debug("getZendeskTickets: error getting tickets for org", "orgName", org.ZendeskOrg.Name, "error", err)
		return nil, fmt.Errorf("getting tickets via zendesk api: %w", err)
	}

	return tickets, nil
}

type NoUserErr struct {
	UserId int64
}

func (e NoUserErr) Error() string {
	return fmt.Sprintf("no user found for id %d", e.UserId)
}

func (m *Model) createBaseTicket(org *orgMigrationDetails, ticket *ticketMigrationDetails) (*psa.Ticket, error) {
	if ticket.ZendeskTicket == nil {
		slog.Debug("createBaseTicket: no zendesk ticket found", "zendeskTicketId", ticket.ZendeskTicket.Id)
		return nil, errors.New("zendesk ticket does not exist")
	}

	var customFields []psa.CustomField
	idField := *m.data.PsaInfo.ZendeskTicketIdField
	idField.Value = ticket.ZendeskTicket.Id
	customFields = append(customFields, idField)
	if ticket.ZendeskTicket.Status == "closed" || ticket.ZendeskTicket.Status == "solved" {
		slog.Debug("createBaseTicket: ticket has closed date", "zendeskTicketId", ticket.ZendeskTicket.Id, "closedOn", ticket.ZendeskTicket.UpdatedAt.In(m.timeZone))
		dateField := *m.data.PsaInfo.ZendeskClosedDateField
		dateField.Value = ticket.ZendeskTicket.UpdatedAt.In(m.timeZone)
		customFields = append(customFields, dateField)
	}

	baseTicket := &psa.Ticket{
		Board:        m.data.PsaInfo.Board,
		Status:       m.data.PsaInfo.StatusOpen,
		Summary:      ticket.ZendeskTicket.Subject,
		Company:      &psa.Company{Id: org.PsaOrg.Id},
		CustomFields: customFields,
	}

	baseTicket.Summary = ticket.ZendeskTicket.Subject
	if len(baseTicket.Summary) > 100 {
		slog.Debug("createBaseTicket: ticket subject is too long", "zendeskTicketId", ticket.ZendeskTicket.Id, "subjectLength", len(ticket.ZendeskTicket.Subject), "psaTicketId", ticket.PsaTicket.Id)
		baseTicket.Summary = baseTicket.Summary[:100]
		baseTicket.InitialInternalAnalysis = fmt.Sprintf("Ticket subject was shortened by migration utility (maximum ticket summary in ConnectWise PSA is 100 characters)\n\n"+
			"Original Subject: %s", ticket.ZendeskTicket.Subject)
	}

	if baseTicket.Summary == "" {
		baseTicket.Summary = "No Subject"
	}

	userString := strconv.Itoa(int(ticket.ZendeskTicket.RequesterId))
	if user, ok := m.data.UsersInPsa[userString]; ok {
		slog.Debug("createBaseTicket: requester is in org data", "zendeskTicketId", ticket.ZendeskTicket.Id, "requesterId", ticket.ZendeskTicket.RequesterId, "psaTicketId", ticket.PsaTicket.Id, "contactId", user.PsaContact.Id)
		baseTicket.Contact = &psa.Contact{Id: user.PsaContact.Id}
	} else {
		slog.Debug("createBaseTicket: requester is not in org data", "zendeskTicketId", ticket.ZendeskTicket.Id, "requesterId", ticket.ZendeskTicket.RequesterId, "psaTicketId", ticket.PsaTicket.Id)
		return nil, NoUserErr{UserId: ticket.ZendeskTicket.RequesterId}
	}

	ownerString := strconv.Itoa(int(ticket.ZendeskTicket.AssigneeId))
	if owner, ok := m.client.Cfg.AgentMappings[ownerString]; ok {
		if owner.PsaId != 0 {
			slog.Debug("createBaseTicket: owner is in agent mappings", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskAssigneeId", ticket.ZendeskTicket.AssigneeId, "psaTicketId", ticket.PsaTicket.Id, "agentId", owner.PsaId)
			baseTicket.Owner = &psa.Owner{Id: owner.PsaId}
		}
	}

	var err error
	baseTicket, err = m.client.CwClient.PostTicket(m.ctx, baseTicket)
	if err != nil {
		slog.Debug("createBaseTicket: error posting base ticket to connectwise", "zendeskTicketId", ticket.ZendeskTicket.Id, "error", err)
		return nil, fmt.Errorf("posting base ticket to connectwise: %w", err)
	}

	return baseTicket, nil
}

func (m *Model) createTicketNotes(ticket *ticketMigrationDetails, comments []zendesk.Comment) error {
	for _, comment := range comments {
		slog.Debug("createTicketNotes: called", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "psaTicketId", ticket.PsaTicket.Id)
		note := &psa.TicketNote{}

		authorString := strconv.Itoa(int(comment.AuthorId))
		if agent, ok := m.client.Cfg.AgentMappings[authorString]; ok {
			note.Member = &psa.Member{Id: agent.PsaId}
		} else if contact, ok := m.data.UsersInPsa[authorString]; ok {
			note.Contact = &psa.Contact{Id: contact.PsaContact.Id}
		} else {
			// check if user is in Zendesk and use it as a label - we aren't making non-selected org users in ConnectWise
			senderName, senderEmail := m.getExternalUserDetails(ticket, comment, authorString)
			note.Text += fmt.Sprintf("**Sent By**: %s (%s)\n", senderName, senderEmail)
		}

		if comment.Public {
			note.DetailDescriptionFlag = true
		} else {
			note.DetailDescriptionFlag = true
			note.InternalAnalysisFlag = true
		}

		note.Text += fmt.Sprintf("**%s**\n", comment.CreatedAt.In(m.timeZone).Format("Mon 1/2/2006 3:04PM"))

		ccs := m.getCcString(&comment)
		if ccs != "" {
			note.Text += fmt.Sprintf("**CCs:** %s\n", ccs)
		}

		note.Text += fmt.Sprintf("\n%s", comment.Body)

		slog.Debug("createTicketNotes: sending post request to create note", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "psaTicketId", ticket.PsaTicket.Id)
		if err := m.client.CwClient.PostTicketNote(m.ctx, ticket.PsaTicket.Id, note); err != nil {
			slog.Error("createTicketNote: error creating note in ticket", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "psaTicketId", ticket.PsaTicket.Id, "error", err)
			return fmt.Errorf("creating note in ticket: %w", err)
		}
	}

	return nil
}

func (m *Model) getExternalUserDetails(ticket *ticketMigrationDetails, comment zendesk.Comment, authorString string) (string, string) {
	slog.Debug("createTicketNotes: author is not in org data", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "authorId", comment.AuthorId, "psaTicketId", ticket.PsaTicket.Id)
	senderName := "Unknown"
	senderEmail := "no email"
	if user, ok := m.data.ExternalUsers[authorString]; !ok {
		user, err := m.client.ZendeskClient.GetUser(m.ctx, comment.AuthorId)
		if err != nil {
		} else {
			m.data.ExternalUsers[authorString] = user
			senderName = user.Name
			if user.Email != "" {
				senderEmail = user.Email
			}
		}

	} else {
		senderName = user.Name
		if user.Email != "" {
			senderEmail = user.Email
		}
	}

	slog.Debug("createTicketNotes: external user details", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "authorId", comment.AuthorId, "psaTicketId", ticket.PsaTicket.Id, "senderName", senderName, "senderEmail", senderEmail)
	return senderName, senderEmail
}

func (m *Model) getCcString(comment *zendesk.Comment) string {
	if comment.Via.Source.To.EmailCcs == nil || len(comment.Via.Source.To.EmailCcs) == 0 {
		slog.Debug("getCcString: no email CCs found in comment", "commentId", comment.Id)
		return ""
	}

	var ccs []string
	slog.Debug("getCcString: processing email CCs", "commentId", comment.Id, "ccCount", len(comment.Via.Source.To.EmailCcs))
	for _, cc := range comment.Via.Source.To.EmailCcs {
		// check if cc is a string
		if cc, ok := cc.(string); ok {
			ccs = append(ccs, cc)
			continue
		}

		var ccString string
		switch v := cc.(type) {
		case int:
			ccString = strconv.Itoa(v)
		case float64:
			ccString = strconv.Itoa(int(v))
		default:
			continue
		}

		if agent, ok := m.client.Cfg.AgentMappings[ccString]; ok {
			ccs = append(ccs, agent.Email)
		} else {
			if contact, ok := m.data.UsersInPsa[ccString]; ok {
				ccs = append(ccs, contact.ZendeskUser.Email)
			}
		}
	}
	return strings.Join(ccs, ", ")
}

func newActiveTicketMigration(orgName string, status ticketStatus) *activeTicketMigration {
	return &activeTicketMigration{orgName: orgName, status: status}
}
