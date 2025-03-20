package zendesk

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type TicketSearchResp struct {
	Tickets []Ticket `json:"results"`
	Meta    Meta     `json:"meta"`
	Links   Links    `json:"links"`
}

type TicketsListResp struct {
	Tickets []Ticket `json:"tickets"`
	Meta    Meta     `json:"meta"`
	Links   Links    `json:"links"`
}

type TicketResp struct {
	Ticket Ticket `json:"ticket"`
}

type Ticket struct {
	Url                  string              `json:"url"`
	Id                   int                 `json:"id"`
	ExternalId           interface{}         `json:"external_id"`
	CreatedAt            time.Time           `json:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at"`
	GeneratedTimestamp   int                 `json:"generated_timestamp"`
	Type                 string              `json:"type"`
	Subject              string              `json:"subject"`
	RawSubject           string              `json:"raw_subject"`
	Description          string              `json:"description"`
	Priority             string              `json:"priority"`
	Status               string              `json:"status"`
	Recipient            interface{}         `json:"recipient"`
	RequesterId          int64               `json:"requester_id"`
	SubmitterId          int64               `json:"submitter_id"`
	AssigneeId           int64               `json:"assignee_id"`
	OrganizationId       int64               `json:"organization_id"`
	GroupId              int64               `json:"group_id"`
	CollaboratorIds      []int64             `json:"collaborator_ids"`
	FollowerIds          []interface{}       `json:"follower_ids"`
	EmailCcIds           []int64             `json:"email_cc_ids"`
	ForumTopicId         interface{}         `json:"forum_topic_id"`
	ProblemId            interface{}         `json:"problem_id"`
	HasIncidents         bool                `json:"has_incidents"`
	IsPublic             bool                `json:"is_public"`
	DueAt                interface{}         `json:"due_at"`
	Tags                 []string            `json:"tags"`
	CustomFields         []TicketCustomField `json:"custom_fields"`
	CustomStatusId       int                 `json:"custom_status_id"`
	EncodedId            string              `json:"encoded_id"`
	FollowupIds          []interface{}       `json:"followup_ids"`
	BrandId              int                 `json:"brand_id"`
	AllowChannelback     bool                `json:"allow_channelback"`
	AllowAttachments     bool                `json:"allow_attachments"`
	FromMessagingChannel bool                `json:"from_messaging_channel"`
}

type TicketCustomField struct {
	Id    int64   `json:"id"`
	Value *string `json:"value"`
}

type TicketCommentsResp struct {
	Comments []Comment `json:"comments"`
	Meta     Meta      `json:"meta"`
	Links    Links     `json:"links"`
}

type Comment struct {
	Id        int64     `json:"id"`
	Type      string    `json:"type"`
	AuthorId  int64     `json:"author_id"`
	Body      string    `json:"body"`
	HtmlBody  string    `json:"html_body"`
	PlainBody string    `json:"plain_body"`
	Public    bool      `json:"public"`
	Via       Via       `json:"via"`
	CreatedAt time.Time `json:"created_at"`
}

type Via struct {
	Channel string `json:"channel"`
	Source  Source `json:"source"`
}
type Source struct {
	From From        `json:"from"`
	To   To          `json:"to"`
	Rel  interface{} `json:"rel"`
}

type From struct {
	Address            string   `json:"address,omitempty"`
	Name               string   `json:"name,omitempty"`
	OriginalRecipients []string `json:"original_recipients,omitempty"`
}

type To struct {
	Name     string  `json:"name"`
	Address  string  `json:"address"`
	EmailCcs []int64 `json:"email_ccs,omitempty"`
}

func (c *Client) GetTicketsWithQuery(ctx context.Context, q SearchQuery, pageSize int, justGetOne bool) ([]Ticket, error) {
	var allTickets []Ticket
	currentPage := &TicketSearchResp{}

	if err := c.exportSearchRequest(ctx, TicketSearchType, q, pageSize, &currentPage); err != nil {
		return nil, fmt.Errorf("an error occured getting the tickets: %w", err)
	}
	slog.Debug("got initial page of tickets", "totalInitialTickets", len(currentPage.Tickets))

	allTickets = append(allTickets, currentPage.Tickets...)

	// used to only return one page - for checking presence of at least one ticket
	if justGetOne {
		return allTickets, nil
	}

	for currentPage.Meta.HasMore {
		nextPage := &TicketSearchResp{}
		if err := c.ApiRequest(ctx, "GET", currentPage.Links.Next, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("an error occured getting next page of tickets: %w", err)
		}
		slog.Debug("got next page of tickets", "totalTicketsInPage", len(nextPage.Tickets))
		allTickets = append(allTickets, nextPage.Tickets...)
		currentPage = nextPage
	}

	slog.Debug("finished getting tickets for org", "totalTicketsFound", len(allTickets))
	return allTickets, nil
}

func (c *Client) GetOrgTickets(ctx context.Context, orgId int64) ([]Ticket, error) {
	initialUrl := fmt.Sprintf("%s/organizations/%d/tickets.json?page[size]=100", c.baseUrl, orgId)
	var allTickets []Ticket
	currentPage := &TicketsListResp{}

	if err := c.ApiRequest(ctx, "GET", initialUrl, nil, &currentPage); err != nil {
		return nil, fmt.Errorf("getting org tickets: %w", err)
	}

	slog.Debug("got initial page of tickets", "orgId", orgId, "totalInitialTickets", len(currentPage.Tickets))
	allTickets = append(allTickets, currentPage.Tickets...)

	for currentPage.Meta.HasMore {
		nextPage := &TicketsListResp{}
		if err := c.ApiRequest(ctx, "GET", currentPage.Links.Next, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("getting next page of org tickets: %w", err)
		}

		slog.Debug("got next page of tickets", "orgId", orgId, "totalTicketsInPage", len(nextPage.Tickets))
		allTickets = append(allTickets, nextPage.Tickets...)
		currentPage = nextPage
	}

	slog.Debug("finished getting tickets for org", "orgId", orgId, "totalTicketsInPage", len(allTickets))
	return allTickets, nil
}

func (c *Client) GetTicket(ctx context.Context, ticketId int64) (*Ticket, error) {
	url := fmt.Sprintf("%s/tickets/%d", c.baseUrl, ticketId)
	t := &TicketResp{}

	if err := c.ApiRequest(ctx, "GET", url, nil, &t); err != nil {
		return nil, fmt.Errorf("getting ticket info: %w", err)
	}

	return &t.Ticket, nil
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
