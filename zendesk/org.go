package zendesk

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"
)

type Organizations struct {
	Organizations []Organization `json:"results"`
}
type Organization struct {
	Organization struct {
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
	} `json:"organization"`
}

func (c *Client) GetOrganizationsWithQuery(ctx context.Context, tags []string) ([]Organization, error) {
	var q string
	var r Organizations
	var orgs []Organization

	if len(tags) > 0 {
		q = "type:organization"
		for _, tag := range tags {
			q += fmt.Sprintf(" tags:%s", tag)
		}
	}

	q = url.QueryEscape(q)
	slog.Debug("GetOrganizationsWithQuery:", "query", q)

	if err := c.searchRequest(ctx, q, &r); err != nil {
		return nil, fmt.Errorf("an error occured getting the organizations: %w", err)
	}

	orgs = r.Organizations

	return orgs, nil
}

func (c *Client) GetOrganization(ctx context.Context, orgId int64) (Organization, error) {
	url := fmt.Sprintf("%s/organizations/%d", c.baseUrl, orgId)
	o := &Organization{}

	if err := c.apiRequest(ctx, "GET", url, nil, &o); err != nil {
		return Organization{}, fmt.Errorf("an error occured getting the organization: %w", err)
	}

	return *o, nil
}
