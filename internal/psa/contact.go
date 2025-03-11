package psa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
)

type ContactsResp []Contact

func (c *Client) PostContact(ctx context.Context, contact *Contact) (*Contact, error) {
	slog.Debug("psa.PostContact called")
	u := fmt.Sprintf("%s/company/contacts", baseUrl)

	jsonBytes, err := json.Marshal(contact)
	if err != nil {
		return nil, fmt.Errorf("marshaling contact to json: %w", err)
	}

	body := bytes.NewReader(jsonBytes)
	respContact := Contact{}

	if err := c.apiRequest(ctx, "POST", u, body, &respContact); err != nil {
		return nil, fmt.Errorf("an error occured creating the contact: %w", err)
	}

	return &respContact, nil
}

func (c *Client) GetContactByEmail(ctx context.Context, email string) (*Contact, error) {
	slog.Debug("psa.GetContactByEmail called")
	query := url.QueryEscape(fmt.Sprintf("communicationItems/type/name=\"email\" AND communicationItems/value=\"%s\"", email))
	u := fmt.Sprintf("%s/company/contacts?childConditions=%s", baseUrl, query)
	contacts := ContactsResp{}

	if err := c.apiRequest(ctx, "GET", u, nil, &contacts); err != nil {
		return nil, fmt.Errorf("an error occured searching for the contact by email: %w", err)
	}

	if len(contacts) == 0 {
		slog.Debug("contact not found", "email", email)
		return nil, nil
	}

	if len(contacts) != 1 {
		return nil, fmt.Errorf("expected 1 contact, got %d", len(contacts))
	}

	return &contacts[0], nil
}
