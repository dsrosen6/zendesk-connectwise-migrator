package zendesk

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
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

func (c *Client) exportSearchRequest(ctx context.Context, searchType SearchType, query SearchQuery, pageSize int, target interface{}) error {
	queryString, err := buildExportSearchQueryString(searchType, query)
	if err != nil {
		return fmt.Errorf("building query string: %w", err)
	}

	u := fmt.Sprintf("%s/search/export.json?filter[type]=%s&query=%s&page[size]=%d", c.baseUrl, searchType, queryString, pageSize)
	if err := c.ApiRequest(ctx, "GET", u, nil, target); err != nil {
		return fmt.Errorf("an error occured searching for the resource: %w", err)
	}

	return nil
}

func (c *Client) searchRequest(ctx context.Context, searchType SearchType, query SearchQuery, target interface{}) error {
	queryString, err := buildSearchQueryString(searchType, query)
	if err != nil {
		return fmt.Errorf("building query string: %w", err)
	}

	u := fmt.Sprintf("%s/search.json?query=%s&count=1000", c.baseUrl, queryString)
	if err := c.ApiRequest(ctx, "GET", u, nil, target); err != nil {
		return fmt.Errorf("an error occured searching for the resource: %w", err)
	}

	return nil
}

func buildExportSearchQueryString(searchType SearchType, query SearchQuery) (string, error) {
	if searchType == "" {
		return "", errors.New("search type cannot be empty")
	}

	var queryParts []string
	if len(query.Tags) > 0 {
		for _, tag := range query.Tags {
			queryParts = append(queryParts, fmt.Sprintf("tags:%s", tag))
		}
	}

	if searchType == TicketSearchType {
		if query.TicketsOrganizationId != 0 {
			queryParts = append(queryParts, fmt.Sprintf("organization:%d", query.TicketsOrganizationId))
		}

		if query.TicketCreatedAfter != (time.Time{}) {
			queryParts = append(queryParts, fmt.Sprintf("created>%s", query.TicketCreatedAfter.Format("2006-01-02")))
		}

		if query.TicketCreatedBefore != (time.Time{}) {
			queryParts = append(queryParts, fmt.Sprintf("created<%s", query.TicketCreatedBefore.Format("2006-01-02")))
		}
	}

	if len(queryParts) == 0 {
		return "", errors.New("no search criteria provided")
	}

	joinedParts := strings.Join(queryParts, " ")
	return url.QueryEscape(joinedParts), nil
}

func buildSearchQueryString(searchType SearchType, query SearchQuery) (string, error) {
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
