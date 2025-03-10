package zendesk

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"
)

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

func (c *Client) GetOrganizationsWithQuery(ctx context.Context, tags []string) ([]Organization, error) {
	slog.Debug("zendesk.Client.GetOrganizationsWithQuery called", "tags", tags)
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
