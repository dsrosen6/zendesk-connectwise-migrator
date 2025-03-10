package zendesk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

type TicketFieldsResp struct {
	TicketFields []TicketField `json:"ticket_fields"`
	Meta         `json:"meta"`
	Links        `json:"links"`
}

type UserFieldsResp struct {
	UserFields []UserField `json:"user_fields"`
	Meta       `json:"meta"`
	Links      `json:"links"`
}

type OrganizationFieldsResp struct {
	OrganizationFields []OrganizationField `json:"organization_fields"`
	Meta               `json:"meta"`
	Links              `json:"links"`
}

type PostTicketField struct {
	TicketField TicketField `json:"ticket_field"`
}

type PostUserField struct {
	UserField UserField `json:"user_field"`
}

type PostOrganizationField struct {
	OrganizationField OrganizationField `json:"organization_field"`
}

type TicketField struct {
	Id               int64  `json:"id"`
	Type             string `json:"type"`
	Title            string `json:"title"`
	AgentDescription string `json:"agent_description"`
	Active           bool   `json:"active"`
}

type UserField struct {
	Id          int64  `json:"id"`
	Type        string `json:"type"`
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
}

type OrganizationField struct {
	Id          int64  `json:"id"`
	Type        string `json:"type"`
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
}

type Meta struct {
	HasMore      bool   `json:"has_more"`
	AfterCursor  string `json:"after_cursor"`
	BeforeCursor string `json:"before_cursor"`
}
type Links struct {
	Prev string `json:"prev"`
	Next string `json:"next"`
}

func (c *Client) PostTicketField(ctx context.Context, fieldType, title, description string) (*TicketField, error) {
	slog.Debug("zendesk.Client.PostTicketField called", "fieldType", fieldType, "title", title, "description", description)
	f := &PostTicketField{
		TicketField: TicketField{
			Type:             fieldType,
			Title:            title,
			AgentDescription: description,
			Active:           true,
		},
	}

	fieldBytes, err := json.Marshal(f)
	if err != nil {
		return nil, fmt.Errorf("marshaling ticket field to json: %w", err)
	}
	body := bytes.NewReader(fieldBytes)

	u := fmt.Sprintf("%s/ticket_fields", c.baseUrl)
	r := &PostTicketField{}

	if err := c.apiRequest(ctx, "POST", u, body, r); err != nil {
		return nil, err
	}

	return &r.TicketField, nil
}

func (c *Client) GetTicketFieldByTitle(ctx context.Context, title string) (*TicketField, error) {
	slog.Debug("zendesk.Client.GetTicketFieldByTitle called", "title", title)
	fields, err := c.GetTicketFields(ctx)
	if err != nil {
		return nil, err
	}

	for _, field := range fields {
		if field.Title == title {
			return &field, nil
		}
	}

	return nil, fmt.Errorf("ticket field with title %s not found", title)
}

func (c *Client) GetTicketFields(ctx context.Context) ([]TicketField, error) {
	slog.Debug("zendesk.Client.GetTicketFields called")
	initialUrl := fmt.Sprintf("%s/ticket_fields?page[size]=100", c.baseUrl)
	allFields := &TicketFieldsResp{}
	currentPage := &TicketFieldsResp{}

	if err := c.apiRequest(ctx, "GET", initialUrl, nil, &currentPage); err != nil {
		return nil, fmt.Errorf("an error occured getting the ticket fields: %w", err)
	}

	allFields.TicketFields = append(allFields.TicketFields, currentPage.TicketFields...)

	for currentPage.Meta.HasMore {
		nextPage := &TicketFieldsResp{}
		if err := c.apiRequest(ctx, "GET", currentPage.Links.Next, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("an error occured getting the ticket fields: %w", err)
		}

		allFields.TicketFields = append(allFields.TicketFields, nextPage.TicketFields...)
		currentPage = nextPage
	}

	return allFields.TicketFields, nil
}

func (c *Client) PostUserField(ctx context.Context, fieldType, key, title, description string) (*UserField, error) {
	slog.Debug("zendesk.Client.PostUserField called", "fieldType", fieldType, "key", key, "title", title, "description", description)
	f := &PostUserField{
		UserField: UserField{
			Type:        fieldType,
			Key:         key,
			Title:       title,
			Description: description,
			Active:      true,
		},
	}

	fieldBytes, err := json.Marshal(f)
	if err != nil {
		return nil, fmt.Errorf("marshaling ticket field to json: %w", err)
	}

	body := bytes.NewReader(fieldBytes)

	u := fmt.Sprintf("%s/user_fields", c.baseUrl)
	r := &PostUserField{}

	if err := c.apiRequest(ctx, "POST", u, body, r); err != nil {
		return nil, err
	}

	return &r.UserField, nil
}

func (c *Client) GetUserFieldByKey(ctx context.Context, key string) (*UserField, error) {
	slog.Debug("zendesk.Client.GetUserFieldByKey called", "key", key)
	fields, err := c.GetUserFields(ctx)
	if err != nil {
		return nil, err
	}

	for _, field := range fields {
		if field.Key == key {
			return &field, nil
		}
	}

	return nil, fmt.Errorf("user field with key %s not found", key)
}

func (c *Client) GetUserFields(ctx context.Context) ([]UserField, error) {
	slog.Debug("zendesk.Client.GetUserFields called")
	initialUrl := fmt.Sprintf("%s/user_fields?page[size]=100", c.baseUrl)
	allFields := &UserFieldsResp{}
	currentPage := &UserFieldsResp{}

	if err := c.apiRequest(ctx, "GET", initialUrl, nil, &currentPage); err != nil {
		return nil, fmt.Errorf("an error occured getting the user fields: %w", err)
	}

	allFields.UserFields = append(allFields.UserFields, currentPage.UserFields...)

	for currentPage.Meta.HasMore {
		nextPage := &UserFieldsResp{}
		if err := c.apiRequest(ctx, "GET", currentPage.Links.Next, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("an error occured getting the user fields: %w", err)
		}

		allFields.UserFields = append(allFields.UserFields, nextPage.UserFields...)
		currentPage = nextPage
	}

	return allFields.UserFields, nil
}

func (c *Client) PostOrgField(ctx context.Context, fieldType, key, title, description string) (*OrganizationField, error) {
	slog.Debug("zendesk.Client.PostOrgField called", "fieldType", fieldType, "key", key, "title", title, "description", description)
	f := &PostOrganizationField{
		OrganizationField: OrganizationField{
			Type:        fieldType,
			Key:         key,
			Title:       title,
			Description: description,
			Active:      true,
		},
	}

	fieldBytes, err := json.Marshal(f)
	if err != nil {
		return nil, fmt.Errorf("marshaling ticket field to json: %w", err)
	}

	body := bytes.NewReader(fieldBytes)

	u := fmt.Sprintf("%s/organization_fields", c.baseUrl)
	r := &PostOrganizationField{}

	if err := c.apiRequest(ctx, "POST", u, body, r); err != nil {
		return nil, err
	}

	return &r.OrganizationField, nil
}

func (c *Client) GetOrgFieldByKey(ctx context.Context, key string) (*OrganizationField, error) {
	slog.Debug("zendesk.Client.GetOrgFieldByKey called", "key", key)
	fields, err := c.GetOrgFields(ctx)
	if err != nil {
		return nil, err
	}

	for _, field := range fields {
		if field.Key == key {
			return &field, nil
		}
	}

	return nil, fmt.Errorf("ticket field with key %s not found", key)
}

func (c *Client) GetOrgFields(ctx context.Context) ([]OrganizationField, error) {
	slog.Debug("zendesk.Client.GetOrgFields called")
	initialUrl := fmt.Sprintf("%s/organization_fields?page[size]=100", c.baseUrl)
	allFields := &OrganizationFieldsResp{}
	currentPage := &OrganizationFieldsResp{}

	if err := c.apiRequest(ctx, "GET", initialUrl, nil, &currentPage); err != nil {
		return nil, fmt.Errorf("an error occured getting the organization fields: %w", err)
	}

	allFields.OrganizationFields = append(allFields.OrganizationFields, currentPage.OrganizationFields...)

	for currentPage.Meta.HasMore {
		nextPage := &OrganizationFieldsResp{}
		if err := c.apiRequest(ctx, "GET", currentPage.Links.Next, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("an error occured getting the organization fields: %w", err)
		}

		allFields.OrganizationFields = append(allFields.OrganizationFields, nextPage.OrganizationFields...)
		currentPage = nextPage
	}

	return allFields.OrganizationFields, nil
}
