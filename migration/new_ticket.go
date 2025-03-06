package migration

import (
	"context"
	"github.com/dsrosen/zendesk-connectwise-migrator/zendesk"

	"github.com/dsrosen/zendesk-connectwise-migrator/cw"
)

func (c *Client) CreateTicket(ctx context.Context, input InputTicket) (cw.Ticket, error) {
	return cw.Ticket{}, nil // TODO: Stuff
}

func (c *Client) MatchOrgToCompany(ctx context.Context, org zendesk.Organization) (cw.Company, error) {
	return c.CwClient.GetCompanyByName(ctx, org.Organization.Name)
}
