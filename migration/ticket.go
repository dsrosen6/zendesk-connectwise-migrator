package migration

import (
	"context"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/cw"
	"github.com/dsrosen/zendesk-connectwise-migrator/zendesk"
	"log/slog"
	"time"
)

type InputTicket struct {
	Subject            string
	InitialDescription string
	Organization       zendesk.Organization
	Requester          zendesk.User
	Assignee           zendesk.User
	Comments           []commentInput
	Closed             bool // ie, "closed"
	ClosedAt           time.Time
}

type commentInput struct {
	Sender    zendesk.User
	Ccs       []zendesk.User
	Body      string
	Public    bool
	CreatedAt time.Time
}

func (c *Client) ConstructInputTicket(ctx context.Context, ticketId int64) (*InputTicket, error) {
	ticketInfo, err := c.ZendeskClient.GetTicket(ctx, ticketId)
	if err != nil {
		return nil, fmt.Errorf("an error occured getting initial ticket info: %w", err)
	}

	inputTicket := &InputTicket{
		Subject:            ticketInfo.Ticket.Subject,
		InitialDescription: ticketInfo.Ticket.Description,
		Closed:             ticketInfo.Ticket.Status == "closed",
		Comments:           []commentInput{},
	}

	inputTicket.Organization, err = c.ZendeskClient.GetOrganization(ctx, ticketInfo.Ticket.OrganizationId)
	if err != nil {
		return nil, fmt.Errorf("an error occured getting ticket Organization: %w", err)
	}

	rawComments, err := c.ZendeskClient.GetAllTicketComments(ctx, ticketId)
	if err != nil {
		return nil, fmt.Errorf("an error occured getting ticket Comments: %w", err)
	}

	inputTicket.Requester, err = c.ZendeskClient.GetUser(ctx, ticketInfo.Ticket.RequesterId)
	if err != nil {
		return nil, fmt.Errorf("an error occured getting ticket Requester: %w", err)
	}

	// don't error - if Assignee is nil, it will be ignored
	inputTicket.Assignee, _ = c.ZendeskClient.GetUser(ctx, ticketInfo.Ticket.AssigneeId)

	for _, comment := range rawComments.Comments {
		ci, err := c.createCommentInput(ctx, comment)
		if err != nil {
			return nil, fmt.Errorf("an error occured creating comment input: %w", err)
		}

		inputTicket.Comments = append(inputTicket.Comments, ci)
	}

	if inputTicket.Closed {
		inputTicket.ClosedAt = ticketInfo.Ticket.UpdatedAt
	}

	slog.Debug("constructed input ticket",
		"subject", inputTicket.Subject,
		"organization", inputTicket.Organization.Organization.Name,
		"requesterName", inputTicket.Requester.User.Name, "requesterEmail", inputTicket.Requester.User.Email, "requesterId", inputTicket.Requester.User.Id,
		"assigneeName", inputTicket.Assignee.User.Name, "assigneeEmail", inputTicket.Assignee.User.Email, "assigneeId", inputTicket.Assignee.User.Id,
		"totalComments", len(inputTicket.Comments),
		"closed", inputTicket.Closed, "closedAt", inputTicket.ClosedAt)
	return inputTicket, nil
}

func (c *Client) createCommentInput(ctx context.Context, comment zendesk.Comment) (commentInput, error) {
	sender, err := c.ZendeskClient.GetUser(ctx, comment.AuthorId)
	if err != nil {
		return commentInput{}, fmt.Errorf("an error occured getting comment author: %w", err)
	}

	var ccs []zendesk.User
	if comment.Via.Source.To.EmailCcs != nil {
		for _, ccId := range comment.Via.Source.To.EmailCcs {
			cc, err := c.ZendeskClient.GetUser(ctx, ccId)
			if err != nil {
				return commentInput{}, fmt.Errorf("an error occured getting comment cc: %w", err)
			}
			ccs = append(ccs, cc)
		}
	}

	return commentInput{
		Sender:    sender,
		Ccs:       ccs,
		Body:      comment.PlainBody,
		Public:    comment.Public,
		CreatedAt: comment.CreatedAt,
	}, nil
}

func (c *Client) CreateTicket(ctx context.Context, input InputTicket) (cw.Ticket, error) {
	return cw.Ticket{}, nil // TODO: Stuff
}
