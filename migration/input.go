package migration

import (
	"context"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/zendesk"
	"time"
)

type InputTicket struct {
	Subject            string
	InitialDescription string
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
	ticketInfo, err := c.zendeskClient.GetTicket(ctx, ticketId)
	if err != nil {
		return nil, fmt.Errorf("an error occured getting initial ticket info: %w", err)
	}

	inputTicket := &InputTicket{
		Subject:            ticketInfo.Ticket.Subject,
		InitialDescription: ticketInfo.Ticket.Description,
		Closed:             ticketInfo.Ticket.Status == "closed",
		Comments:           []commentInput{},
	}

	rawComments, err := c.zendeskClient.GetAllTicketComments(ctx, ticketId)
	if err != nil {
		return nil, fmt.Errorf("an error occured getting ticket Comments: %w", err)
	}

	inputTicket.Requester, err = c.zendeskClient.GetUser(ctx, ticketInfo.Ticket.RequesterId)
	if err != nil {
		return nil, fmt.Errorf("an error occured getting ticket Requester: %w", err)
	}

	// don't error - if Assignee is nil, it will be ignored
	inputTicket.Assignee, _ = c.zendeskClient.GetUser(ctx, ticketInfo.Ticket.AssigneeId)

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

	return inputTicket, nil
}

func (c *Client) createCommentInput(ctx context.Context, comment zendesk.Comment) (commentInput, error) {
	sender, err := c.zendeskClient.GetUser(ctx, comment.AuthorId)
	if err != nil {
		return commentInput{}, fmt.Errorf("an error occured getting comment author: %w", err)
	}

	var ccs []zendesk.User
	if comment.Via.Source.To.EmailCcs != nil {
		for _, ccId := range comment.Via.Source.To.EmailCcs {
			cc, err := c.zendeskClient.GetUser(ctx, ccId)
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
