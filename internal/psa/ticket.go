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

	if _, err := c.ApiRequest(ctx, "GET", url, nil, &t); err != nil {
		return nil, fmt.Errorf("getting the ticket: %w", err)
	}

	return t, nil
}

func (c *Client) PostTicket(ctx context.Context, ticket *Ticket) (*Ticket, error) {
	url := fmt.Sprintf("%s/service/tickets", baseUrl)

	ticketBytes, err := json.Marshal(ticket)
	if err != nil {
		return nil, fmt.Errorf("marshaling the ticket to json: %w", err)
	}

	body := bytes.NewReader(ticketBytes)
	t := &Ticket{}

	if _, err := c.ApiRequest(ctx, "POST", url, body, t); err != nil {
		return nil, fmt.Errorf("posting the ticket: %w", err)
	}

	return t, nil
}

func (c *Client) PostTicketNote(ctx context.Context, ticketId int, note *TicketNote) error {
	url := fmt.Sprintf("%s/service/tickets/%d/notes", baseUrl, ticketId)
	
	noteBytes, err := json.Marshal(note)
	if err != nil {
		return fmt.Errorf("marshaling ticket note to json: %w", err)
	}

	body := bytes.NewReader(noteBytes)

	if _, err := c.ApiRequest(ctx, "POST", url, body, nil); err != nil {
		return fmt.Errorf("posting the ticket note: %w", err)
	}

	return nil
}
