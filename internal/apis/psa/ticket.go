package psa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

func (c *Client) GetTicket(ctx context.Context, ticketId int) (Ticket, error) {
	slog.Debug("psa.GetTicket called")
	url := fmt.Sprintf("%s/service/tickets/%d", baseUrl, ticketId)
	t := &Ticket{}

	if err := c.apiRequest(ctx, "GET", url, nil, &t); err != nil {
		return Ticket{}, fmt.Errorf("an error occured getting the ticket: %w", err)
	}

	return *t, nil
}

func (c *Client) PostTicket(ctx context.Context, ticket Ticket) (Ticket, error) {
	slog.Debug("psa.PostTicket called")
	url := fmt.Sprintf("%s/service/tickets", baseUrl)

	ticketBytes, err := json.Marshal(ticket)
	if err != nil {
		return Ticket{}, fmt.Errorf("an error occured marshaling the ticket to json: %w", err)
	}

	body := bytes.NewReader(ticketBytes)
	t := &Ticket{}

	if err := c.apiRequest(ctx, "POST", url, body, t); err != nil {
		return Ticket{}, fmt.Errorf("an error occured posting the ticket: %w", err)
	}

	return *t, nil
}
