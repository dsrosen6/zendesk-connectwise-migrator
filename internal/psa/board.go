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

func (c *Client) GetBoardStatuses(ctx context.Context, boardId int) ([]Status, error) {
	url := fmt.Sprintf("%s/service/boards/%d/statuses", baseUrl, boardId)
	var b []Status

	// TODO: Handle Pagination
	if _, err := c.ApiRequest(ctx, "GET", url, nil, &b); err != nil {
		return nil, fmt.Errorf("an error occured getting board statuses: %w", err)
	}

	return b, nil
}
