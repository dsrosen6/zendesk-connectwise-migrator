package psa

import (
	"context"
	"fmt"
	"net/url"
)

type CompaniesResp []Company

func (c *Client) GetCompanyByName(ctx context.Context, name string) (Company, error) {
	query := url.QueryEscape(fmt.Sprintf("name=\"%s\"", name))
	u := fmt.Sprintf("%s/company/companies?conditions=%s", baseUrl, query)
	co := CompaniesResp{}

	if err := c.apiRequest(ctx, "GET", u, nil, &co); err != nil {
		return Company{}, fmt.Errorf("an error occured getting the company: %w", err)
	}

	if len(co) != 1 {
		return Company{}, fmt.Errorf("expected 1 company, got %d", len(co))
	}

	return co[0], nil
}
