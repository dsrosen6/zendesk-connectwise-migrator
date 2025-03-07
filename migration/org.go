package migration

import (
	"context"
	"github.com/dsrosen/zendesk-connectwise-migrator/cw"
	"github.com/dsrosen/zendesk-connectwise-migrator/zendesk"
)

func (c *Client) MatchZdOrgToCwCompany(ctx context.Context, org zendesk.Organization) (cw.Company, error) {
	return c.CwClient.GetCompanyByName(ctx, org.Organization.Name)
}
