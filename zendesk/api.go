package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	zendeskApiUrl = "zendesk.com/api/v2"
)

type Client struct {
	creds      *Creds
	baseUrl    string
	httpClient *http.Client
}

type Creds struct {
	token     string
	username  string
	subdomain string
}

func NewClient(creds *Creds) *Client {
	creds.username = fmt.Sprintf("%s/token", creds.username)
	return &Client{
		creds:      creds,
		baseUrl:    fmt.Sprintf("https://%s.%s", creds.subdomain, zendeskApiUrl),
		httpClient: &http.Client{},
	}
}

func (c *Client) apiRequest(ctx context.Context, method, url string, body io.Reader, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("an error occured creating the request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.creds.username, c.creds.token)

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
		return fmt.Errorf("non-success status code: %d", res.StatusCode)
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
