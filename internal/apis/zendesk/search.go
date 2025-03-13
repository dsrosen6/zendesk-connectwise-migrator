package zendesk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"
)

type SearchQuery struct {
	Tags                  []string
	TicketsOrganizationId int64
	TicketCreatedAfter    time.Time
	TicketCreatedBefore   time.Time
}

type SearchType string

const (
	TicketSearchType SearchType = "ticket"
	OrgSearchType    SearchType = "organization"
	UserSearchType   SearchType = "user"
)

func (c *Client) searchRequest(ctx context.Context, searchType SearchType, query SearchQuery, target interface{}) error {
	queryString, err := buildQueryString(searchType, query)
	if err != nil {
		return fmt.Errorf("building query string: %w", err)
	}

	slog.Debug("zendesk.Client.searchRequest called", "query", query)
	u := fmt.Sprintf("%s/search.json?query=%s&count=1000", c.baseUrl, queryString)
	if err := c.apiRequest(ctx, "GET", u, nil, target); err != nil {
		return fmt.Errorf("an error occured searching for the resource: %w", err)
	}

	return nil
}

func buildQueryString(searchType SearchType, query SearchQuery) (string, error) {
	if searchType == "" {
		return "", errors.New("search type cannot be empty")
	}

	qs := fmt.Sprintf("type:%s", searchType)

	if len(query.Tags) > 0 {
		for _, tag := range query.Tags {
			qs += fmt.Sprintf(" tags:%s", tag)
		}
	}

	if searchType == TicketSearchType {
		if query.TicketsOrganizationId != 0 {
			qs += fmt.Sprintf(" organization:%d", query.TicketsOrganizationId)
		}

		if query.TicketCreatedAfter != (time.Time{}) {
			qs += fmt.Sprintf(" created>%s", query.TicketCreatedAfter.Format("2006-01-02"))
		}

		if query.TicketCreatedBefore != (time.Time{}) {
			qs += fmt.Sprintf(" created<%s", query.TicketCreatedBefore.Format("2006-01-02"))
		}
	}

	return url.QueryEscape(qs), nil
}
