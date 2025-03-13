package zendesk

import (
	"context"
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
	slog.Debug("zendesk.Client.GetOrganizationsWithQuery called", "query", q)

	var allOrgs []Organization
	currentPage := &OrgSearchResp{}

	if err := c.searchRequest(ctx, OrgSearchType, q, &currentPage); err != nil {
		return nil, fmt.Errorf("an error occured getting the organizations: %w", err)
	}

	allOrgs = append(allOrgs, currentPage.Organizations...)

	for currentPage.NextPage != "" {
		nextPage := &OrgSearchResp{}
		if err := c.apiRequest(ctx, "GET", currentPage.NextPage, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("an error occured getting next page of organizations: %w", err)
		}

		allOrgs = append(allOrgs, nextPage.Organizations...)
		currentPage = nextPage
	}

	slog.Debug("returning orgs", "totalOrgs", len(allOrgs))
	return allOrgs, nil
}

func (c *Client) GetOrganization(ctx context.Context, orgId int64) (Organization, error) {
	slog.Debug("zendesk.Client.GetOrganization called", "orgId", orgId)
	u := fmt.Sprintf("%s/organizations/%d", c.baseUrl, orgId)
	var r struct {
		Organization Organization `json:"organization"`
	}

	if err := c.apiRequest(ctx, "GET", u, nil, &r); err != nil {
		return Organization{}, fmt.Errorf("an error occured getting the organization: %w", err)
	}

	return r.Organization, nil
}
