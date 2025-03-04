package migration

import (
	"context"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/zendesk"
	"time"
)

type InputTicket struct {
	subject            string
	initialDescription string
	requester          zendesk.User
	assignee           zendesk.User
	comments           []commentInput
	status             string // ie, "closed"
}

type commentInput struct {
	sender    zendesk.User
	ccs       []zendesk.User
	body      string
	public    bool
	createdAt time.Time
}

func (c *Client) ConstructInputTicket(ctx context.Context, ticketId int64) (*InputTicket, error) {
	ticketInfo, err := c.zendeskClient.GetTicket(ctx, ticketId)
	if err != nil {
		return nil, fmt.Errorf("an error occured getting initial ticket info: %w", err)
	}

	rawComments, err := c.zendeskClient.GetAllTicketComments(ctx, ticketId)
	if err != nil {
		return nil, fmt.Errorf("an error occured getting ticket comments: %w", err)
	}

	requester, err := c.zendeskClient.GetUser(ctx, ticketInfo.Ticket.RequesterId)
	if err != nil {
		return nil, fmt.Errorf("an error occured getting ticket requester: %w", err)
	}

	// don't error - if assignee is nil, it will be ignored
	assignee, _ := c.zendeskClient.GetUser(ctx, ticketInfo.Ticket.AssigneeId)

	var comments []commentInput
	for _, comment := range rawComments.Comments {
		ci, err := c.createCommentInput(ctx, comment)
		if err != nil {
			return nil, fmt.Errorf("an error occured creating comment input: %w", err)
		}
		comments = append(comments, ci)
	}

	return &InputTicket{
		subject:            ticketInfo.Ticket.Subject,
		initialDescription: ticketInfo.Ticket.Description,
		requester:          requester,
		assignee:           assignee,
		comments:           comments,
		status:             ticketInfo.Ticket.Status,
	}, nil
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
		sender:    sender,
		ccs:       ccs,
		body:      comment.PlainBody,
		public:    comment.Public,
		createdAt: comment.CreatedAt,
	}, nil
}
