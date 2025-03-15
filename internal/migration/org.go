package migration

import (
	"context"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/apis/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/apis/zendesk"
)

func (c *Client) MatchZdOrgToCwCompany(ctx context.Context, org zendesk.Organization) (psa.Company, error) {
	comp, err := c.CwClient.GetCompanyByName(ctx, org.Name)
	if err != nil {
		return psa.Company{}, err
	}

	return comp, nil
}
