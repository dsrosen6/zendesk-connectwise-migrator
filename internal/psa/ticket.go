package psa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
)

type PatchPayload []PatchOperation

type PatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value"`
}

func (c *Client) GetTickets(ctx context.Context, childConditionQuery *string) ([]Ticket, error) {
	q := ""
	if childConditionQuery != nil {
		q = "&customFieldConditions=" + url.QueryEscape(*childConditionQuery)
	}

	u := fmt.Sprintf("%s/service/tickets?page=1&pageSize=100%s", baseUrl, q)
	var allTickets []Ticket
	var currentPage []Ticket
	var pagination PaginationDetails

	if p, err := c.ApiRequest(ctx, "GET", u, nil, &currentPage); err != nil {
		return nil, fmt.Errorf("getting tickets: %w", err)
	} else {
		pagination = p
	}

	allTickets = append(allTickets, currentPage...)

	for pagination.HasMorePages && pagination.NextLink != "" {
		var nextPage []Ticket
		if p, err := c.ApiRequest(ctx, "GET", pagination.NextLink, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("getting next page of tickets: %w", err)
		} else {
			pagination = p
		}

		allTickets = append(allTickets, nextPage...)
		currentPage = nextPage
	}

	return allTickets, nil
}

func (c *Client) GetTicket(ctx context.Context, ticketId int) (*Ticket, error) {
	u := fmt.Sprintf("%s/service/tickets/%d", baseUrl, ticketId)
	t := &Ticket{}

	if _, err := c.ApiRequest(ctx, "GET", u, nil, &t); err != nil {
		return nil, fmt.Errorf("getting the ticket: %w", err)
	}

	return t, nil
}

func (c *Client) PostTicket(ctx context.Context, ticket *Ticket) (*Ticket, error) {
	u := fmt.Sprintf("%s/service/tickets", baseUrl)

	ticketBytes, err := json.Marshal(ticket)
	slog.Debug("ticket creation payload created", "payload", string(ticketBytes))
	if err != nil {
		return nil, fmt.Errorf("marshaling the ticket to json: %w", err)
	}

	body := bytes.NewReader(ticketBytes)
	t := &Ticket{}

	if _, err := c.ApiRequest(ctx, "POST", u, body, t); err != nil {
		return nil, fmt.Errorf("posting the ticket: %w", err)
	}

	return t, nil
}

func (c *Client) UpdateTicketStatus(ctx context.Context, ticket *Ticket, newStatusId int) error {
	u := fmt.Sprintf("%s/service/tickets/%d", baseUrl, ticket.Id)

	payload := PatchPayload{
		{
			Op:    "replace",
			Path:  "status/id",
			Value: newStatusId,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	slog.Debug("ticket status update payload created", "payload", string(payloadBytes))
	if err != nil {
		return fmt.Errorf("marshaling patch operation to json: %w", err)
	}

	body := bytes.NewReader(payloadBytes)

	if _, err := c.ApiRequest(ctx, "PATCH", u, body, nil); err != nil {
		return fmt.Errorf("updating the ticket status: %w", err)
	}

	return nil
}

func (c *Client) PostTicketNote(ctx context.Context, ticketId int, note *TicketNote) error {
	u := fmt.Sprintf("%s/service/tickets/%d/notes", baseUrl, ticketId)

	noteBytes, err := json.Marshal(note)
	if err != nil {
		return fmt.Errorf("marshaling ticket note to json: %w", err)
	}

	body := bytes.NewReader(noteBytes)

	if _, err := c.ApiRequest(ctx, "POST", u, body, nil); err != nil {
		return fmt.Errorf("posting the ticket note: %w", err)
	}

	return nil
}
