package migration

import (
	"context"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
)

func (c *Client) MatchZdOrgToCwCompany(ctx context.Context, org *zendesk.Organization) (*psa.Company, error) {
	comp, err := c.CwClient.GetCompanyByName(ctx, org.Name)
	if err != nil {
		return nil, err
	}

	return comp, nil
}
