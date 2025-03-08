package zendesk

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

type Organization struct {
	Url                string      `json:"url"`
	Id                 int64       `json:"id"`
	Name               string      `json:"name"`
	SharedTickets      bool        `json:"shared_tickets"`
	SharedComments     bool        `json:"shared_comments"`
	ExternalId         interface{} `json:"external_id"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
	DomainNames        []string    `json:"domain_names"`
	Details            string      `json:"details"`
	Notes              string      `json:"notes"`
	GroupId            interface{} `json:"group_id"`
	Tags               []string    `json:"tags"`
	OrganizationFields struct {
	} `json:"organization_fields"`
}

type OrganizationFieldsResp struct {
	OrganizationFields []OrganizationField `json:"organization_fields"`
	Meta               struct {
		HasMore      bool   `json:"has_more"`
		AfterCursor  string `json:"after_cursor"`
		BeforeCursor string `json:"before_cursor"`
	} `json:"meta"`
	Links struct {
		Prev string `json:"prev"`
		Next string `json:"next"`
	} `json:"links"`
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

func (c *Client) GetOrganizationsWithQuery(ctx context.Context, tags []string) ([]Organization, error) {
	var q string
	var r struct {
		Organizations []Organization `json:"results"`
	}

	var orgs []Organization

	if len(tags) > 0 {
		q = "type:organization"
		for _, tag := range tags {
			q += fmt.Sprintf(" tags:%s", tag)
		}
	}

	q = url.QueryEscape(q)

	if err := c.searchRequest(ctx, q, &r); err != nil {
		return nil, fmt.Errorf("an error occured getting the organizations: %w", err)
	}

	orgs = r.Organizations

	return orgs, nil
}

func (c *Client) GetOrganization(ctx context.Context, orgId int64) (Organization, error) {
	u := fmt.Sprintf("%s/organizations/%d", c.baseUrl, orgId)
	var r struct {
		Organization Organization `json:"organization"`
	}

	if err := c.apiRequest(ctx, "GET", u, nil, &r); err != nil {
		return Organization{}, fmt.Errorf("an error occured getting the organization: %w", err)
	}

	return r.Organization, nil
}

func (c *Client) GetOrgCustomFields(ctx context.Context) ([]OrganizationField, error) {
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
