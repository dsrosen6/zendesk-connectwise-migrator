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

type NoUserFoundErr struct{}

func (e NoUserFoundErr) Error() string {
	return "No user was found with the provided email"
}

func (c *Client) PostContact(ctx context.Context, payload *ContactPostBody) (*Contact, error) {
	u := fmt.Sprintf("%s/company/contacts", baseUrl)

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling contact to json: %w", err)
	}

	body := bytes.NewReader(jsonBytes)
	respContact := Contact{}

	if _, err := c.ApiRequest(ctx, "POST", u, body, &respContact); err != nil {
		slog.Error("failed to post post new connectwise contact", "body", string(jsonBytes))
		return nil, fmt.Errorf("an error occured creating the contact: %w", err)
	}

	return &respContact, nil
}

func (c *Client) GetContactByEmail(ctx context.Context, email string) (*Contact, error) {
	query := url.QueryEscape(fmt.Sprintf("communicationItems/type/name=\"email\" AND communicationItems/value=\"%s\"", email))
	u := fmt.Sprintf("%s/company/contacts?childConditions=%s", baseUrl, query)
	contacts := ContactsResp{}

	if _, err := c.ApiRequest(ctx, "GET", u, nil, &contacts); err != nil {
		return nil, fmt.Errorf("an error occured searching for the contact by email: %w", err)
	}

	if len(contacts) == 0 {
		return nil, NoUserFoundErr{}
	}

	if len(contacts) != 1 {
		return nil, fmt.Errorf("expected 1 contact, got %d", len(contacts))
	}

	return &contacts[0], nil
}
