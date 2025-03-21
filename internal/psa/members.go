package psa

import (
	"context"
	"fmt"
)

func (c *Client) GetMembers(ctx context.Context) ([]Member, error) {
	url := fmt.Sprintf("%s/system/members?page=1&pageSize=100", baseUrl)
	var allMembers []Member
	var currentPage []Member
	var pagination PaginationDetails

	if p, err := c.ApiRequest(ctx, "GET", url, nil, &currentPage); err != nil {
		return nil, fmt.Errorf("getting psa members: %w", err)
	} else {
		pagination = p
	}

	allMembers = append(allMembers, currentPage...)

	for pagination.HasMorePages && pagination.NextLink != "" {
		var nextPage []Member
		if p, err := c.ApiRequest(ctx, "GET", pagination.NextLink, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("getting psa members: %w", err)
		} else {
			pagination = p
		}

		allMembers = append(allMembers, nextPage...)
		currentPage = nextPage
	}

	return allMembers, nil
}
