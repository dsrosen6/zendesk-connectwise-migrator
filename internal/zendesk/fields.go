package zendesk

import (
	"context"
	"fmt"
	"time"
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

type TicketField struct {
	Url                 string      `json:"url"`
	Id                  int64       `json:"id"`
	Type                string      `json:"type"`
	Title               string      `json:"title"`
	RawTitle            string      `json:"raw_title"`
	Description         string      `json:"description"`
	RawDescription      string      `json:"raw_description"`
	Position            int         `json:"position"`
	Active              bool        `json:"active"`
	Required            bool        `json:"required"`
	CollapsedForAgents  bool        `json:"collapsed_for_agents"`
	RegexpForValidation *string     `json:"regexp_for_validation"`
	TitleInPortal       string      `json:"title_in_portal"`
	RawTitleInPortal    string      `json:"raw_title_in_portal"`
	VisibleInPortal     bool        `json:"visible_in_portal"`
	EditableInPortal    bool        `json:"editable_in_portal"`
	RequiredInPortal    bool        `json:"required_in_portal"`
	AgentCanEdit        bool        `json:"agent_can_edit"`
	Tag                 interface{} `json:"tag"`
	CreatedAt           time.Time   `json:"created_at"`
	UpdatedAt           time.Time   `json:"updated_at"`
	Removable           bool        `json:"removable"`
	Key                 interface{} `json:"key"`
	AgentDescription    *string     `json:"agent_description"`
	SystemFieldOptions  []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"system_field_options,omitempty"`
	SubTypeId          int `json:"sub_type_id,omitempty"`
	CustomFieldOptions []struct {
		Id      int64  `json:"id"`
		Name    string `json:"name"`
		RawName string `json:"raw_name"`
		Value   string `json:"value"`
		Default bool   `json:"default"`
	} `json:"custom_field_options,omitempty"`
	CustomStatuses []struct {
		Url                string    `json:"url"`
		Id                 int64     `json:"id"`
		StatusCategory     string    `json:"status_category"`
		AgentLabel         string    `json:"agent_label"`
		EndUserLabel       string    `json:"end_user_label"`
		Description        string    `json:"description"`
		EndUserDescription string    `json:"end_user_description"`
		Active             bool      `json:"active"`
		Default            bool      `json:"default"`
		CreatedAt          time.Time `json:"created_at"`
		UpdatedAt          time.Time `json:"updated_at"`
	} `json:"custom_statuses,omitempty"`
}

type UserField struct {
	Url                    string      `json:"url"`
	Id                     int64       `json:"id"`
	Type                   string      `json:"type"`
	Key                    string      `json:"key"`
	Title                  string      `json:"title"`
	Description            string      `json:"description"`
	RawTitle               string      `json:"raw_title"`
	RawDescription         string      `json:"raw_description"`
	Position               int         `json:"position"`
	Active                 bool        `json:"active"`
	System                 bool        `json:"system"`
	RegexpForValidation    interface{} `json:"regexp_for_validation"`
	CreatedAt              time.Time   `json:"created_at"`
	UpdatedAt              time.Time   `json:"updated_at"`
	RelationshipTargetType string      `json:"relationship_target_type,omitempty"`
	RelationshipFilter     struct {
		All []struct {
			Field    string `json:"field"`
			Operator string `json:"operator"`
			Value    string `json:"value"`
		} `json:"all"`
		Any []interface{} `json:"any"`
	} `json:"relationship_filter,omitempty"`
}

type OrganizationField struct {
	Url                 string      `json:"url"`
	Id                  int64       `json:"id"`
	Type                string      `json:"type"`
	Key                 string      `json:"key"`
	Title               string      `json:"title"`
	Description         string      `json:"description"`
	RawTitle            string      `json:"raw_title"`
	RawDescription      string      `json:"raw_description"`
	Position            int         `json:"position"`
	Active              bool        `json:"active"`
	System              bool        `json:"system"`
	RegexpForValidation interface{} `json:"regexp_for_validation"`
	CreatedAt           time.Time   `json:"created_at"`
	UpdatedAt           time.Time   `json:"updated_at"`
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

func (c *Client) GetTicketFields(ctx context.Context) ([]TicketField, error) {
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

func (c *Client) GetUserFields(ctx context.Context) ([]UserField, error) {
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

func (c *Client) GetOrgFields(ctx context.Context) ([]OrganizationField, error) {
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
