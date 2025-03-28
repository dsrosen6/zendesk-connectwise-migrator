package zendesk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

const (
	zendeskApiUrl = "zendesk.com/api/v2"
)

type Client struct {
	creds      Creds
	baseUrl    string
	httpClient *http.Client
}

type Creds struct {
	Token     string `mapstructure:"token" json:"token"`
	Username  string `mapstructure:"username" json:"username"`
	Subdomain string `mapstructure:"subdomain" json:"subdomain"`
}

type Meta struct {
	HasMore bool `json:"has_more"`
}
type Links struct {
	Next string `json:"next"`
}

func NewClient(creds Creds, httpClient *http.Client) *Client {
	creds.Username = fmt.Sprintf("%s/token", creds.Username)
	return &Client{
		creds:      creds,
		baseUrl:    fmt.Sprintf("https://%s.%s", creds.Subdomain, zendeskApiUrl),
		httpClient: httpClient,
	}
}

func (c *Client) ConnectionTest(ctx context.Context) error {
	url := fmt.Sprintf("%s/users?page[size]=1", c.baseUrl)

	u := &UsersResp{}
	if err := c.ApiRequest(ctx, "GET", url, nil, &u); err != nil {
		return errors.New("failed to connect to Zendesk API")
	}

	return nil
}

// ApiRequest is a wrapper for apiRequest, meant for more streamlined error logging.
func (c *Client) ApiRequest(ctx context.Context, method, url string, body io.Reader, target interface{}) error {
	if err := c.apiRequest(ctx, method, url, body, target); err != nil {
		slog.Debug("zendesk api error", "error", err)
		return fmt.Errorf("running Zendesk API request: %w", err)
	}

	return nil
}

func (c *Client) apiRequest(ctx context.Context, method, url string, body io.Reader, target interface{}) error {
	const maxRetries = 3
	var retryAfter int

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			slog.Debug("sending zendesk API request", "method", method, "url", url, "attempt", attempt)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return fmt.Errorf("an error occured creating the request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(c.creds.Username, c.creds.Token)

		res, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("an error occured sending the request: %w", err)
		}

		if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated {
			data, err := io.ReadAll(res.Body)
			if err != nil {
				return fmt.Errorf("an error occured reading the response body: %w", err)
			}

			if target != nil {
				if err := json.Unmarshal(data, target); err != nil {
					return fmt.Errorf("an error occured unmarshaling the response to JSON: %w", err)
				}
			}

			return nil
		}

		if res.StatusCode == http.StatusTooManyRequests {
			retryAfterHeader := res.Header.Get("Retry-After")

			if retryAfterHeader != "" {
				retryAfter, err = strconv.Atoi(retryAfterHeader)
				if err != nil {
					slog.Debug("failed to parse Retry-After header", "header", retryAfterHeader)
					retryAfter = 1
				}

			} else {
				slog.Debug("missing Retry-After header")
				retryAfter = 1
			}

			slog.Debug("zendesk rate limit exceeded, retrying",
				"retryAfter", retryAfter,
				"totalRetries", fmt.Sprintf("%d/%d", attempt, maxRetries))
		} else {
			retryAfter = 5
			slog.Debug("zendesk API request failed - waiting 5 seconds if retries remain", "statusCode", res.StatusCode,
				"totalRetries", fmt.Sprintf("%d/%d", attempt, maxRetries))
		}

		err = res.Body.Close()
		if err != nil {
			return err
		}
		time.Sleep(time.Duration(retryAfter) * time.Second)
	}

	return fmt.Errorf("max retries exceeded")
}
