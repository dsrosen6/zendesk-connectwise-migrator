package psa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

func (c *Client) GetTicket(ctx context.Context, ticketId int) (*Ticket, error) {
	url := fmt.Sprintf("%s/service/tickets/%d", baseUrl, ticketId)
	t := &Ticket{}

	if err := c.ApiRequest(ctx, "GET", url, nil, &t); err != nil {
		return nil, fmt.Errorf("an error occured getting the ticket: %w", err)
	}

	return t, nil
}

func (c *Client) PostTicket(ctx context.Context, ticket *Ticket) (*Ticket, error) {
	url := fmt.Sprintf("%s/service/tickets", baseUrl)

	ticketBytes, err := json.Marshal(ticket)
	if err != nil {
		return nil, fmt.Errorf("an error occured marshaling the ticket to json: %w", err)
	}

	body := bytes.NewReader(ticketBytes)
	t := &Ticket{}

	if err := c.ApiRequest(ctx, "POST", url, body, t); err != nil {
		return nil, fmt.Errorf("an error occured posting the ticket: %w", err)
	}

	return t, nil
}

func (c *Client) PostTicketNote(ctx context.Context, ticket *Ticket, note *TicketNote) (*TicketNote, error) {
	return nil, nil
}
