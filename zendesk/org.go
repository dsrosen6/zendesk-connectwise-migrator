package zendesk

import (
	"context"
	"fmt"
	"time"
)

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

func (c *Client) GetOrganization(ctx context.Context, orgId int64) (Organization, error) {
	url := fmt.Sprintf("%s/organizations/%d", c.baseUrl, orgId)
	o := &Organization{}

	if err := c.apiRequest(ctx, "GET", url, nil, &o); err != nil {
		return Organization{}, fmt.Errorf("an error occured getting the organization: %w", err)
	}

	return *o, nil
}
