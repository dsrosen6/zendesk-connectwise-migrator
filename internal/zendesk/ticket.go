package zendesk

import (
	"context"
	"fmt"
	"time"
)

type TicketSearchResp struct {
	Tickets []Ticket `json:"results"`
	Meta    Meta     `json:"meta"`
	Links   Links    `json:"links"`
}

type Ticket struct {
	Id          int       `json:"id"`
	UpdatedAt   time.Time `json:"updated_at"`
	Subject     string    `json:"subject"`
	Status      string    `json:"status"`
	RequesterId int64     `json:"requester_id"`
	AssigneeId  int64     `json:"assignee_id"`
}

type TicketCommentsResp struct {
	Comments []Comment `json:"comments"`
	Meta     Meta      `json:"meta"`
	Links    Links     `json:"links"`
}

type Comment struct {
	Id        int64     `json:"id"`
	AuthorId  int64     `json:"author_id"`
	Body      string    `json:"body"`
	Public    bool      `json:"public"`
	CreatedAt time.Time `json:"created_at"`
	Via       struct {
		Source struct {
			To struct {
				EmailCcs []any
			} `json:"to"`
		} `json:"source"`
	} `json:"via"`
}

func (c *Client) GetTicketsWithQuery(ctx context.Context, q SearchQuery, pageSize int, limit int) ([]Ticket, error) {
	var allTickets []Ticket
	currentPage := &TicketSearchResp{}

	if err := c.exportSearchRequest(ctx, TicketSearchType, q, pageSize, &currentPage); err != nil {
		return nil, fmt.Errorf("an error occured getting the tickets: %w", err)
	}

	allTickets = append(allTickets, currentPage.Tickets...)

	for currentPage.Meta.HasMore {
		if limit > 0 && len(allTickets) >= limit {
			break
		}

		nextPage := &TicketSearchResp{}
		if err := c.ApiRequest(ctx, "GET", currentPage.Links.Next, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("an error occured getting next page of tickets: %w", err)
		}
		allTickets = append(allTickets, nextPage.Tickets...)
		currentPage = nextPage
	}

	return allTickets, nil
}

func (c *Client) GetAllTicketComments(ctx context.Context, ticketId int64) ([]Comment, error) {
	initialUrl := fmt.Sprintf("%s/tickets/%d/comments.json?page[size]=100", c.baseUrl, ticketId)
	var allComments []Comment
	currentPage := &TicketCommentsResp{}

	if err := c.ApiRequest(ctx, "GET", initialUrl, nil, &currentPage); err != nil {
		return nil, fmt.Errorf("an error occured getting initial ticket comments: %w", err)
	}

	// Append the first page of comments to the allComments slice
	allComments = append(allComments, currentPage.Comments...)

	for currentPage.Meta.HasMore {
		nextPage := &TicketCommentsResp{}
		if err := c.ApiRequest(ctx, "GET", currentPage.Links.Next, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("an error occured getting next page of ticket comments: %w", err)
		}

		allComments = append(allComments, nextPage.Comments...)
		currentPage = nextPage

	}

	return allComments, nil
}
