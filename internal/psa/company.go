package psa

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
)

type CompaniesResp []Company

func (c *Client) GetCompanyByName(ctx context.Context, name string) (*Company, error) {
	query := url.QueryEscape(fmt.Sprintf("name=\"%s\"", name))
	u := fmt.Sprintf("%s/company/companies?conditions=%s", baseUrl, query)
	cos := CompaniesResp{}

	if _, err := c.ApiRequest(ctx, "GET", u, nil, &cos); err != nil {
		return nil, fmt.Errorf("an error occured getting the company: %w", err)
	}

	if len(cos) != 1 {
		slog.Warn("matching companies - does not equal 1 company", "totalCompanies", len(cos))
		return nil, fmt.Errorf("expected 1 company, got %d", len(cos))
	}

	return &cos[0], nil
}
