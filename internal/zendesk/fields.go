package zendesk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

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

type PostUserField struct {
	UserField UserField `json:"user_field"`
}

type PostOrganizationField struct {
	OrganizationField OrganizationField `json:"organization_field"`
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

func (c *Client) PostUserField(ctx context.Context, fieldType, key, title, description string) (*UserField, error) {
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

	if err := c.ApiRequest(ctx, "POST", u, body, r); err != nil {
		return nil, err
	}

	return &r.UserField, nil
}

func (c *Client) GetUserFieldByKey(ctx context.Context, key string) (*UserField, error) {
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
	initialUrl := fmt.Sprintf("%s/user_fields?page[size]=100", c.baseUrl)
	allFields := &UserFieldsResp{}
	currentPage := &UserFieldsResp{}

	if err := c.ApiRequest(ctx, "GET", initialUrl, nil, &currentPage); err != nil {
		return nil, fmt.Errorf("an error occured getting the user fields: %w", err)
	}

	allFields.UserFields = append(allFields.UserFields, currentPage.UserFields...)

	for currentPage.Meta.HasMore {
		nextPage := &UserFieldsResp{}
		if err := c.ApiRequest(ctx, "GET", currentPage.Links.Next, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("an error occured getting the user fields: %w", err)
		}

		allFields.UserFields = append(allFields.UserFields, nextPage.UserFields...)
		currentPage = nextPage
	}

	return allFields.UserFields, nil
}

func (c *Client) PostOrgField(ctx context.Context, fieldType, key, title, description string) (*OrganizationField, error) {
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

	if err := c.ApiRequest(ctx, "POST", u, body, r); err != nil {
		return nil, err
	}

	return &r.OrganizationField, nil
}

func (c *Client) GetOrgFieldByKey(ctx context.Context, key string) (*OrganizationField, error) {
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
	initialUrl := fmt.Sprintf("%s/organization_fields?page[size]=100", c.baseUrl)
	allFields := &OrganizationFieldsResp{}
	currentPage := &OrganizationFieldsResp{}

	if err := c.ApiRequest(ctx, "GET", initialUrl, nil, &currentPage); err != nil {
		return nil, fmt.Errorf("an error occured getting the organization fields: %w", err)
	}

	allFields.OrganizationFields = append(allFields.OrganizationFields, currentPage.OrganizationFields...)

	for currentPage.Meta.HasMore {
		nextPage := &OrganizationFieldsResp{}
		if err := c.ApiRequest(ctx, "GET", currentPage.Links.Next, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("an error occured getting the organization fields: %w", err)
		}

		allFields.OrganizationFields = append(allFields.OrganizationFields, nextPage.OrganizationFields...)
		currentPage = nextPage
	}

	return allFields.OrganizationFields, nil
}
