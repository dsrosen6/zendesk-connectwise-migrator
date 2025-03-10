package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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

	u := &Users{}
	if err := c.apiRequest(ctx, "GET", url, nil, &u); err != nil {
		return err
	}

	return nil
}

func (c *Client) searchRequest(ctx context.Context, query string, target interface{}) error {
	u := fmt.Sprintf("%s/search.json?query=%s", c.baseUrl, query)
	if err := c.apiRequest(ctx, "GET", u, nil, target); err != nil {
		return fmt.Errorf("an error occured searching for the resource: %w", err)
	}

	return nil
}

func (c *Client) apiRequest(ctx context.Context, method, url string, body io.Reader, target interface{}) error {
	slog.Debug("sending zendesk api request", "method", method, "url", url, "body", body)
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

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(res.Body)

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		slog.Error("zendesk api request failed", "status_code", res.StatusCode, "url", url, "body", body)
		return fmt.Errorf("status code: %s", res.Status)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("an error occured reading the response body: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("an error occured unmarshaling the response to JSON: %w", err)
	}

	return nil
}
