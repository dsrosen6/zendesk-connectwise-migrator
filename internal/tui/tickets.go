package tui

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"strconv"
	"strings"
)

type ticketMigrationDetails struct {
	ZendeskTicket *zendesk.Ticket `json:"zendesk_ticket"`
	PsaTicket     *psa.Ticket     `json:"psa_ticket"`

	Migrated bool `json:"migrated"`
}

type fatalErrMsg struct {
	Msg string
	Err error
}

func (m *RootModel) runTicketMigration(org *orgMigrationDetails) tea.Cmd {
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
			if psaId, ok := m.data.TicketsInPsa[strconv.Itoa(ticket.Id)]; !ok {
				td := &ticketMigrationDetails{
					ZendeskTicket: &ticket,
					PsaTicket:     &psa.Ticket{},
				}

				slog.Debug("ticket needs to be migrated", "zendeskId", ticket.Id)
				ticketsToMigrate = append(ticketsToMigrate, td)
				m.ticketsToMigrate++

			} else {
				slog.Debug("ticket already migrated", "zendeskId", ticket.Id, "psaId", psaId)
			}
		}

		for _, ticket := range ticketsToMigrate {
			if m.client.Cfg.TestLimit > 0 && m.ticketsMigrated >= m.client.Cfg.TestLimit {
				slog.Info("testLimit reached")
				m.ticketOrgsMigrated++
				return nil
			}

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

			if err := m.createTicketNotes(ticket, comments); err != nil {
				slog.Error("creating comments for connectwise ticket", "orgName", org.ZendeskOrg.Name, "ticketId", ticket.PsaTicket.Id, "error", err)
				m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("creating comments on ticket %d: %w", ticket.PsaTicket.Id, err)))
				m.totalErrors++
				continue
			}

			if ticket.ZendeskTicket.Status == "closed" || ticket.ZendeskTicket.Status == "solved" {
				slog.Debug("closing ticket", "closedOn", ticket.ZendeskTicket.UpdatedAt, "closedBy", ticket.PsaTicket.Owner.Identifier)
				if err := m.client.CwClient.UpdateTicketStatus(ctx, ticket.PsaTicket, m.data.PsaInfo.StatusClosed.Id); err != nil {
					slog.Error("error closing ticket", "orgName", org.ZendeskOrg.Name, "zendeskTicketId", ticket.ZendeskTicket.Id, "psaTicketId", ticket.PsaTicket.Id, "error", err)
					m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("closing %s psa ticket %d: %w", org.ZendeskOrg.Name, ticket.PsaTicket.Id, err)))
					m.totalErrors++
					continue
				}
			}

			m.data.writeToOutput(goodGreenOutput("CREATED", fmt.Sprintf("migrated ticket: %d", ticket.ZendeskTicket.Id)))
			m.data.TicketsInPsa[strconv.Itoa(ticket.ZendeskTicket.Id)] = ticket.PsaTicket.Id
			m.ticketsMigrated++
		}

		m.ticketOrgsMigrated++
		return nil
	}
}

func (m *RootModel) getAlreadyMigrated() tea.Cmd {
	return func() tea.Msg {
		s := fmt.Sprintf("id=%d AND value != null", m.data.PsaInfo.ZendeskTicketIdField.Id)
		tickets, err := m.client.CwClient.GetTickets(ctx, &s)
		if err != nil {
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
						m.ticketsMigrated++
						break
					}
				}
			}
		}

		return switchStatusMsg(migratingTickets)
	}
}

func (m *RootModel) getZendeskTickets(org *orgMigrationDetails) ([]zendesk.Ticket, error) {
	slog.Info("getting tickets for org", "orgName", org.ZendeskOrg.Name)
	q := zendesk.SearchQuery{
		TicketsOrganizationId: org.ZendeskOrg.Id,
		TicketCreatedAfter:    org.Tag.StartDate,
		TicketCreatedBefore:   org.Tag.EndDate,
	}

	tickets, err := m.client.ZendeskClient.GetTicketsWithQuery(ctx, q, 100, m.client.Cfg.TestLimit)
	if err != nil {
		slog.Error("getting tickets for org", "orgName", org.ZendeskOrg.Name, "error", err)
		return nil, fmt.Errorf("getting tickets via zendesk api: %w", err)
	}

	return tickets, nil
}

func (m *RootModel) createBaseTicket(org *orgMigrationDetails, ticket *ticketMigrationDetails) (*psa.Ticket, error) {
	if ticket.ZendeskTicket == nil {
		return nil, errors.New("zendesk ticket does not exist")
	}

	var customFields []psa.CustomField
	idField := *m.data.PsaInfo.ZendeskTicketIdField
	idField.Value = ticket.ZendeskTicket.Id
	customFields = append(customFields, idField)
	if ticket.ZendeskTicket.Status == "closed" || ticket.ZendeskTicket.Status == "solved" {
		slog.Debug("ticket has closed date", "zendeskTicketId", ticket.ZendeskTicket.Id, "closedOn", ticket.ZendeskTicket.UpdatedAt)
		dateField := *m.data.PsaInfo.ZendeskClosedDateField
		dateField.Value = ticket.ZendeskTicket.UpdatedAt
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
		baseTicket.Summary = baseTicket.Summary[:100]
		baseTicket.InitialInternalAnalysis = fmt.Sprintf("Ticket subject was shortened by migration utility (maximum ticket summary in ConnectWise PSA is 100 characters)\n\n"+
			"Original Subject: %s", ticket.ZendeskTicket.Subject)
	}

	userString := strconv.Itoa(int(ticket.ZendeskTicket.RequesterId))
	if user, ok := m.data.UsersInPsa[userString]; ok {
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

func (m *RootModel) createTicketNotes(ticket *ticketMigrationDetails, comments []zendesk.Comment) error {
	for _, comment := range comments {
		note := &psa.TicketNote{}

		authorString := strconv.Itoa(int(comment.AuthorId))
		slog.Debug("author id", "authorId", authorString)
		if agent, ok := m.client.Cfg.AgentMappings[authorString]; ok {
			slog.Debug("author is in agent mappings", "authorId", authorString)
			note.Member = &psa.Member{Id: agent.PsaId}
		} else if contact, ok := m.data.UsersInPsa[authorString]; ok {
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

		ccs := m.getCcString(&comment)
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

func (m *RootModel) getCcString(comment *zendesk.Comment) string {
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
				if contact, ok := m.data.UsersInPsa[ccString]; ok {
					slog.Debug("cc is in org data", "id", ccString, "email", contact.ZendeskUser.Email)
					ccs = append(ccs, contact.ZendeskUser.Email)
				}
			}
		}
	}
	return strings.Join(ccs, ", ")
}
