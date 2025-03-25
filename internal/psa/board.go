package psa

import (
	"context"
	"fmt"
)

func (c *Client) GetBoards(ctx context.Context) ([]Board, error) {
	url := fmt.Sprintf("%s/service/boards", baseUrl)
	var b []Board

	// TODO: Handle Pagination
	if _, err := c.ApiRequest(ctx, "GET", url, nil, &b); err != nil {
		return nil, fmt.Errorf("an error occured getting boards: %w", err)
	}

	return b, nil
}

func (c *Client) GetBoardTypes(ctx context.Context, boardId int) ([]BoardType, error) {
	url := fmt.Sprintf("%s/service/boards/%d/types?page=1&pageSize=100", baseUrl, boardId)
	var allTypes []BoardType
	var currentPage []BoardType
	var pagination PaginationDetails

	if p, err := c.ApiRequest(ctx, "GET", url, nil, &currentPage); err != nil {
		return nil, fmt.Errorf("getting board types: %w", err)
	} else {
		pagination = p
	}

	allTypes = append(allTypes, currentPage...)

	for pagination.HasMorePages && pagination.NextLink != "" {
		var nextPage []BoardType
		if p, err := c.ApiRequest(ctx, "GET", pagination.NextLink, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("getting board types: %w", err)
		} else {
			pagination = p
		}

		allTypes = append(allTypes, nextPage...)
		currentPage = nextPage
	}

	return allTypes, nil
}

func (c *Client) GetBoardStatuses(ctx context.Context, boardId int) ([]Status, error) {
	url := fmt.Sprintf("%s/service/boards/%d/statuses", baseUrl, boardId)
	var b []Status

	// TODO: Handle Pagination
	if _, err := c.ApiRequest(ctx, "GET", url, nil, &b); err != nil {
		return nil, fmt.Errorf("an error occured getting board statuses: %w", err)
	}

	return b, nil
}
