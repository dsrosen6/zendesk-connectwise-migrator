package migration

import (
	"context"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/apis/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/apis/zendesk"
	"log/slog"
)

func (c *Client) MatchZdOrgToCwCompany(ctx context.Context, org zendesk.Organization) (psa.Company, error) {
	slog.Debug("migration.MatchZdOrgToCwCompany called")
	comp, err := c.CwClient.GetCompanyByName(ctx, org.Name)
	if err != nil {
		slog.Error("error getting company by name", "name", org.Name, "error", err)
		return psa.Company{}, err
	}

	return comp, nil
}
