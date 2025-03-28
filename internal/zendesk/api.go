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

type RateLimitErr struct{}

func (r RateLimitErr) Error() string {
	return "rate limit exceeded"
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
	slog.Debug("zendesk.apiRequest: called", "method", method, "url", url)
	const maxRetries = 3
	var retryAfter int

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			slog.Debug("zendesk.apiRequest: making additional attempt", "method", method, "url", url, "attempt", attempt)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			slog.Debug("zendesk.apiRequest: error creating request", "method", method, "url", url, "error", err)
			return fmt.Errorf("an error occured creating the request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(c.creds.Username, c.creds.Token)

		res, err := c.httpClient.Do(req)
		if err != nil {
			slog.Debug("zendesk.apiRequest: error sending request", "method", method, "url", url, "error", err)
			return fmt.Errorf("an error occured sending the request: %w", err)
		}

		closeErr := func() error {
			defer res.Body.Close()
			if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated {
				data, err := io.ReadAll(res.Body)
				if err != nil {
					slog.Debug("zendesk.apiRequest: error reading response body", "error", err)
					return fmt.Errorf("an error occured reading the response body: %w", err)
				}

				if target != nil {
					if err := json.Unmarshal(data, target); err != nil {
						slog.Debug("zendesk.apiRequest: error unmarshaling response", "data", string(data), "error", err)
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
						slog.Debug("zendesk.apiRequest: error parsing Retry-After header", "headerValue", retryAfterHeader, "error", err)
						retryAfter = 1
					}

				} else {
					slog.Debug("zendesk.apiRequest: Retry-After header not set, defaulting to 1 second")
					retryAfter = 1
				}

				slog.Debug("zendesk.apiRequest: rate limit exceeded, retrying after", "retryAfter", retryAfter, "attempt", attempt)
				return RateLimitErr{}
			}
			retryAfter = 5
			slog.Debug("zendesk.apiRequest: received non-200 response", "method", method, "url", url, "statusCode", res.StatusCode)
			return fmt.Errorf("received non-200 response: %s (status code: %d)", res.Status, res.StatusCode)
		}()

		if closeErr == nil {
			return nil
		}

		if !errors.As(closeErr, &RateLimitErr{}) {
			slog.Debug("zendesk.apiRequest: non-rate limit error encountered", "error", closeErr)
			return closeErr
		}

		time.Sleep(time.Duration(retryAfter) * time.Second)
	}

	slog.Debug("zendesk.apiRequest: max retries exceeded", "method", method, "url", url)
	return fmt.Errorf("max retries exceeded")
}
