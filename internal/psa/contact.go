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

type Contact struct {
	Id        int                `json:"id"`
	FirstName string             `json:"firstName"`
	LastName  string             `json:"lastName"`
	Company   ContactCompanyInfo `json:"company"`
	Site      struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			SiteHref   string `json:"site_href"`
			MobileGuid string `json:"mobileGuid"`
		} `json:"_info"`
	} `json:"site"`
	InactiveFlag       bool   `json:"inactiveFlag"`
	Title              string `json:"title,omitempty"`
	MarriedFlag        bool   `json:"marriedFlag"`
	ChildrenFlag       bool   `json:"childrenFlag"`
	UnsubscribeFlag    bool   `json:"unsubscribeFlag"`
	MobileGuid         string `json:"mobileGuid"`
	DefaultPhoneType   string `json:"defaultPhoneType,omitempty"`
	DefaultPhoneNbr    string `json:"defaultPhoneNbr,omitempty"`
	DefaultBillingFlag bool   `json:"defaultBillingFlag"`
	DefaultFlag        bool   `json:"defaultFlag"`
	CompanyLocation    struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			LocationHref string `json:"location_href"`
		} `json:"_info"`
	} `json:"companyLocation"`
	CommunicationItems []CommunicationItem `json:"communicationItems"`
	Types              []interface{}       `json:"types"`
}

type ContactCompanyInfo struct {
	Id         int    `json:"id"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
}

type CommunicationItem struct {
	Id                int                   `json:"id"`
	Type              CommunicationItemType `json:"type"`
	Value             string                `json:"value"`
	DefaultFlag       bool                  `json:"defaultFlag"`
	Domain            string                `json:"domain,omitempty"`
	CommunicationType string                `json:"communicationType"`
}
type CommunicationItemType struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

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
