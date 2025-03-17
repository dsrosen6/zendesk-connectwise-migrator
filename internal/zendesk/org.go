package zendesk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

type OrgSearchResp struct {
	Organizations []Organization `json:"results"`
	NextPage      string         `json:"next_page"`
	PreviousPage  string         `json:"previous_page"`
	Count         int            `json:"count"`
}

type Organization struct {
	Url                string             `json:"url"`
	Id                 int64              `json:"id"`
	Name               string             `json:"name"`
	SharedTickets      bool               `json:"shared_tickets"`
	SharedComments     bool               `json:"shared_comments"`
	ExternalId         interface{}        `json:"external_id"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
	DomainNames        []string           `json:"domain_names"`
	Details            string             `json:"details"`
	Notes              string             `json:"notes"`
	GroupId            interface{}        `json:"group_id"`
	Tags               []string           `json:"tags"`
	OrganizationFields OrganizationFields `json:"organization_fields"`
}

type OrganizationFields struct {
	PSACompanyId int64 `json:"psa_company"`
}

func (c *Client) GetOrganizationsWithQuery(ctx context.Context, q SearchQuery) ([]Organization, error) {
	var allOrgs []Organization
	currentPage := &OrgSearchResp{}

	if err := c.searchRequest(ctx, OrgSearchType, q, &currentPage); err != nil {
		return nil, fmt.Errorf("getting the organizations: %w", err)
	}

	allOrgs = append(allOrgs, currentPage.Organizations...)

	for currentPage.NextPage != "" {
		nextPage := &OrgSearchResp{}
		if err := c.ApiRequest(ctx, "GET", currentPage.NextPage, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("getting next page of organizations: %w", err)
		}

		allOrgs = append(allOrgs, nextPage.Organizations...)
		currentPage = nextPage
	}

	return allOrgs, nil
}

func (c *Client) GetOrganization(ctx context.Context, orgId int64) (Organization, error) {
	u := fmt.Sprintf("%s/organizations/%d", c.baseUrl, orgId)
	var r struct {
		Organization Organization `json:"organization"`
	}

	if err := c.ApiRequest(ctx, "GET", u, nil, &r); err != nil {
		return Organization{}, fmt.Errorf("getting the organization: %w", err)
	}

	return r.Organization, nil
}

func (c *Client) UpdateOrganization(ctx context.Context, org *Organization) (*Organization, error) {
	slog.Debug("updating organization", "orgId", org.Id, "fields", org.OrganizationFields)
	u := fmt.Sprintf("%s/organizations/%d", c.baseUrl, org.Id)
	jsonBytes, err := json.Marshal(org)
	if err != nil {
		return nil, fmt.Errorf("marshaling organization to json: %w", err)
	}

	body := bytes.NewReader(jsonBytes)

	if err := c.ApiRequest(ctx, "PUT", u, body, org); err != nil {
		return nil, fmt.Errorf("updating the organization: %w", err)
	}

	return org, nil
}
