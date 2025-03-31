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

type fatalErrMsg struct {
	Msg string
	Err error
}

func (m *Model) runTicketMigration(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		m.currentOrgMigration = &currentOrgDetails{
			orgName:          org.ZendeskOrg.Name,
			status:           ticketStatusGetting,
			ticketsToProcess: 0,
			ticketsProcessed: 0,
		}

		zTickets, err := m.getZendeskTickets(org)
		if err != nil {
			m.ticketMigrationErrors++
			slog.Error("getting tickets for org", "orgName", org.ZendeskOrg.Name, "error", err)
			m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("couldn't get zendesk tickets for org %s: %s", org.ZendeskOrg.Name, err)), errOutput)
			return nil
		}

		m.currentOrgMigration.ticketsProcessed = org.TicketsAlreadyInPSA
		m.currentOrgMigration.ticketsToProcess = len(zTickets)
		m.currentOrgMigration.status = ticketStatusMigrating

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
						m.currentOrgMigration.ticketsProcessed++
						slog.Warn("creating base ticket: no user found", "orgName", org.ZendeskOrg.Name, "zendeskTicketId", ticket.ZendeskTicket.Id, "userId", noUserErr.UserId)
						m.writeToOutput(warnYellowOutput("WARN", fmt.Sprintf("couldn't create %s ticket %d to psa ticket: no user found", org.ZendeskOrg.Name, ticket.ZendeskTicket.Id)), errOutput)
						return
					}

					slog.Error("creating base ticket", "orgName", org.ZendeskOrg.Name, "zendeskTicketId", ticket.ZendeskTicket.Id, "error", err)
					m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("couldn't create %s ticket %d to psa ticket: %s", org.ZendeskOrg.Name, ticket.ZendeskTicket.Id, err)), errOutput)
					m.ticketMigrationErrors++
					m.ticketsProcessed++
					m.currentOrgMigration.ticketsProcessed++
					return
				}

				comments, err := m.client.ZendeskClient.GetAllTicketComments(m.ctx, int64(ticket.ZendeskTicket.Id))
				if err != nil {
					slog.Error("getting comments for zendesk ticket", "orgName", org.ZendeskOrg.Name, "zendeskTicketId", ticket.ZendeskTicket.Id, "error", err)
					m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("getting comments for %s ticket %d", org.ZendeskOrg.Name, ticket.ZendeskTicket.Id)), errOutput)
					m.ticketMigrationErrors++
					m.ticketsProcessed++
					m.currentOrgMigration.ticketsProcessed++
					return
				}

				slog.Debug("creating ticket notes", "zendeskId", ticket.ZendeskTicket.Id, "psaId", ticket.PsaTicket.Id)
				if err := m.createTicketNotes(ticket, comments); err != nil {
					slog.Error("creating comments for connectwise ticket", "orgName", org.ZendeskOrg.Name, "zendeskTicketId", ticket.ZendeskTicket.Id, "psaTicketId", ticket.PsaTicket.Id, "error", err)
					m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("couldn't create comments on %s ticket %d: %s", org.ZendeskOrg.Name, ticket.PsaTicket.Id, err)), errOutput)
					m.ticketMigrationErrors++
					m.ticketsProcessed++
					m.currentOrgMigration.ticketsProcessed++
					return
				}

				if ticket.ZendeskTicket.Status == "closed" || ticket.ZendeskTicket.Status == "solved" {
					slog.Debug("runTicketMigration: closing ticket", "closedOn", ticket.ZendeskTicket.UpdatedAt)
					if err := m.client.CwClient.UpdateTicketStatus(m.ctx, ticket.PsaTicket, m.data.PsaInfo.StatusClosed.Id); err != nil {
						slog.Error("closing ticket", "orgName", org.ZendeskOrg.Name, "zendeskTicketId", ticket.ZendeskTicket.Id, "psaTicketId", ticket.PsaTicket.Id, "error", err)
						m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("couldn't close %s psa ticket %d: %s", org.ZendeskOrg.Name, ticket.PsaTicket.Id, err)), errOutput)
						m.ticketMigrationErrors++
						m.ticketsProcessed++
						m.currentOrgMigration.ticketsProcessed++
						return
					}
				}

				slog.Debug("runTicketMigration: migration complete for ticket", "orgName", org.ZendeskOrg.Name, "zendeskTicketId", ticket.ZendeskTicket.Id, "psaTicketId", ticket.PsaTicket.Id)
				m.mu.Lock()
				m.data.TicketsInPsa[strconv.Itoa(ticket.ZendeskTicket.Id)] = ticket.PsaTicket.Id
				m.mu.Unlock()
				m.newTicketsCreated++
				m.ticketsProcessed++
				m.currentOrgMigration.ticketsProcessed++
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

		for _, ticket := range tickets {
			for _, field := range ticket.CustomFields {
				if field.Id == m.data.PsaInfo.ZendeskTicketIdField.Id {
					// if value is an int, it's a zendesk ticket id
					if _, ok := field.Value.(float64); ok {
						val := strconv.Itoa(int(field.Value.(float64)))
						m.data.TicketsInPsa[val] = ticket.Id
						m.ticketsProcessed++
						for _, org := range m.data.AllOrgs {
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
		slog.Debug("createBaseTicket: posting base ticket to connectwise", "zendeskTicketId", ticket.ZendeskTicket.Id, "error", err)
		return nil, fmt.Errorf("posting base ticket to connectwise: %w", err)
	}

	return baseTicket, nil
}

func (m *Model) createTicketNotes(ticket *ticketMigrationDetails, comments []zendesk.Comment) error {
	for _, comment := range comments {
		slog.Debug("creating ticket note", "zendeskTicketId", ticket.ZendeskTicket.Id, "commentId", comment.Id)
		note := &psa.TicketNote{}

		authorString := strconv.Itoa(int(comment.AuthorId))
		if agent, ok := m.client.Cfg.AgentMappings[authorString]; ok {
			slog.Debug("createTicketNotes: author is in agent mappings", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "authorId", comment.AuthorId, "psaTicketId", ticket.PsaTicket.Id, "agentId", agent.PsaId)
			note.Member = &psa.Member{Id: agent.PsaId}
		} else if contact, ok := m.data.UsersInPsa[authorString]; ok {
			slog.Debug("createTicketNotes: author is in org data", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "authorId", comment.AuthorId, "psaTicketId", ticket.PsaTicket.Id, "contactId", contact.PsaContact.Id)
			note.Contact = &psa.Contact{Id: contact.PsaContact.Id}
		} else {
			// check if user is in Zendesk and use it as a label - we aren't making non-selected org users in ConnectWise
			senderName, senderEmail := m.getExternalUserDetails(ticket, comment, authorString)
			note.Text += fmt.Sprintf("**Sent By**: %s (%s)\n", senderName, senderEmail)
		}

		if comment.Public {
			slog.Debug("createTicketNotes: comment is public", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "psaTicketId", ticket.PsaTicket.Id)
			note.DetailDescriptionFlag = true
		} else {
			slog.Debug("createTicketNotes: comment is private", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "psaTicketId", ticket.PsaTicket.Id)
			note.DetailDescriptionFlag = true
			note.InternalAnalysisFlag = true
		}

		slog.Debug("createTicketNotes: adding note created at timestamp", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "psaTicketId", ticket.PsaTicket.Id)
		note.Text += fmt.Sprintf("**%s**\n", comment.CreatedAt.In(m.timeZone).Format("Mon 1/2/2006 3:04PM"))

		slog.Debug("createTicketNotes: checking for email CCs", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "psaTicketId", ticket.PsaTicket.Id)
		ccs := m.getCcString(&comment)
		if ccs != "" {
			slog.Debug("createTicketNotes: adding ccs to note", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "psaTicketId", ticket.PsaTicket.Id, "ccs", ccs)
			note.Text += fmt.Sprintf("**CCs:** %s\n", ccs)
		}

		slog.Debug("createTicketNotes: adding comment body to note", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "psaTicketId", ticket.PsaTicket.Id)
		note.Text += fmt.Sprintf("\n%s", comment.Body)

		slog.Debug("createTicketNotes: sending post request to create note", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "psaTicketId", ticket.PsaTicket.Id)
		if err := m.client.CwClient.PostTicketNote(m.ctx, ticket.PsaTicket.Id, note); err != nil {
			slog.Error("creating note in ticket", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "psaTicketId", ticket.PsaTicket.Id, "error", err)
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
			slog.Warn("createTicketNotes: couldn't find zendesk user for comment - using default label")
		} else {
			m.data.ExternalUsers[authorString] = user
			slog.Debug("createTicketNotes: found non-org zendesk user via api for comment", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "authorId", comment.AuthorId, "psaTicketId", ticket.PsaTicket.Id, "userName", user.Name, "userEmail", user.Email)
			senderName = user.Name
			if user.Email != "" {
				senderEmail = user.Email
			}
		}

	} else {
		slog.Debug("createTicketNotes: found non-org zendesk user in data for comment", "zendeskTicketId", ticket.ZendeskTicket.Id, "zendeskCommentId", comment.Id, "authorId", comment.AuthorId, "psaTicketId", ticket.PsaTicket.Id, "userName", user.Name, "userEmail", user.Email)
		senderName = user.Name
		if user.Email != "" {
			senderEmail = user.Email
		}
	}
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
			slog.Debug("getCcString: cc is a string", "commentId", comment.Id, "ccEmail", cc)
			ccs = append(ccs, cc)
			continue
		}

		var ccString string
		switch v := cc.(type) {
		case int:
			slog.Debug("getCcString: cc is an int", "commentId", comment.Id, "ccId", v)
			ccString = strconv.Itoa(v)
		case float64:
			slog.Debug("getCcString: cc is a float64", "commentId", comment.Id, "ccId", v)
			ccString = strconv.Itoa(int(v))
		default:
			continue
		}

		if agent, ok := m.client.Cfg.AgentMappings[ccString]; ok {
			slog.Debug("getCcString: cc is in agent mappings", "commentId", comment.Id, "ccId", ccString, "email", agent.Email)
			ccs = append(ccs, agent.Email)
		} else {
			if contact, ok := m.data.UsersInPsa[ccString]; ok {
				slog.Debug("getCcString: cc is in org data", "commentId", comment.Id, "ccId", ccString, "email", contact.ZendeskUser.Email)
				ccs = append(ccs, contact.ZendeskUser.Email)
			}
		}
	}
	return strings.Join(ccs, ", ")
}
