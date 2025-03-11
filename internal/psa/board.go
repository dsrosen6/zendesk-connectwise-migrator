package psa

import (
	"context"
	"fmt"
	"log/slog"
)

func (c *Client) GetBoards(ctx context.Context) ([]Board, error) {
	slog.Debug("psa.GetBoards called")
	url := fmt.Sprintf("%s/service/boards", baseUrl)
	var b []Board

	if err := c.apiRequest(ctx, "GET", url, nil, &b); err != nil {
		return nil, fmt.Errorf("an error occured getting boards: %w", err)
	}

	return b, nil
}

func (c *Client) GetBoardStatuses(ctx context.Context, boardId int) ([]Status, error) {
	slog.Debug("psa.GetBoardStatuses called")
	url := fmt.Sprintf("%s/boards/%d/statuses", baseUrl, boardId)
	var b []Status
	if err := c.apiRequest(ctx, "GET", url, nil, &b); err != nil {
		return nil, fmt.Errorf("an error occured getting board statuses: %w", err)
	}

	return b, nil
}
