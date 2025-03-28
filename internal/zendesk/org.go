package zendesk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

type OrgSearchResp struct {
	Organizations []Organization `json:"results"`
	NextPage      string         `json:"next_page"`
}

type Organization struct {
	Id                 int64  `json:"id"`
	Name               string `json:"name"`
	OrganizationFields struct {
		PSACompanyId int64 `json:"psa_company"`
	} `json:"organization_fields"`
}

func (c *Client) GetOrganizationsWithQuery(ctx context.Context, q SearchQuery) ([]Organization, error) {
	var allOrgs []Organization
	currentPage := &OrgSearchResp{}

	if err := c.searchRequest(ctx, OrgSearchType, q, &currentPage); err != nil {
		return nil, fmt.Errorf("getting the organizations: %w", err)
	}

	allOrgs = append(allOrgs, currentPage.Organizations...)

	for currentPage.NextPage != "" {
		nextPage := &OrgSearchResp{}
		if err := c.ApiRequest(ctx, "GET", currentPage.NextPage, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("getting next page of organizations: %w", err)
		}

		allOrgs = append(allOrgs, nextPage.Organizations...)
		currentPage = nextPage
	}

	return allOrgs, nil
}

func (c *Client) GetOrganization(ctx context.Context, orgId int64) (Organization, error) {
	u := fmt.Sprintf("%s/organizations/%d", c.baseUrl, orgId)
	var r struct {
		Organization Organization `json:"organization"`
	}

	if err := c.ApiRequest(ctx, "GET", u, nil, &r); err != nil {
		return Organization{}, fmt.Errorf("getting the organization: %w", err)
	}

	return r.Organization, nil
}

func (c *Client) UpdateOrganization(ctx context.Context, org *Organization) (*Organization, error) {
	u := fmt.Sprintf("%s/organizations/%d", c.baseUrl, org.Id)

	b := &struct {
		Organization *Organization `json:"organization"`
	}{Organization: org}

	jsonBytes, err := json.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("marshaling organization to json: %w", err)
	}

	body := bytes.NewReader(jsonBytes)

	if err := c.ApiRequest(ctx, "PUT", u, body, org); err != nil {
		return nil, fmt.Errorf("updating the organization: %w", err)
	}

	return org, nil
}
