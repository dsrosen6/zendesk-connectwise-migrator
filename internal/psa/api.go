package psa

import (
	"context"
	"encoding/base64"
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
	baseUrl = "https://api-na.myconnectwise.net/v4_6_release/apis/3.0"
)

type Client struct {
	encodedCreds string
	clientId     string
	httpClient   *http.Client
}

type Creds struct {
	CompanyId  string `mapstructure:"company_id" json:"company_id"`
	PublicKey  string `mapstructure:"public_key" json:"public_key"`
	PrivateKey string `mapstructure:"private_key" json:"private_key"`
	ClientId   string `mapstructure:"client_id" json:"client_id"`
}

func NewClient(creds Creds, httpClient *http.Client) *Client {
	username := fmt.Sprintf("%s+%s", creds.CompanyId, creds.PublicKey)

	return &Client{
		encodedCreds: basicAuth(username, creds.PrivateKey),
		clientId:     creds.ClientId,
		httpClient:   httpClient,
	}
}

func (c *Client) ConnectionTest(ctx context.Context) error {
	url := fmt.Sprintf("%s/company/companies?pageSize=1", baseUrl)
	co := CompaniesResp{}

	if err := c.ApiRequest(ctx, "GET", url, nil, &co); err != nil {
		return errors.New("failed to connect to Connectwise API")
	}

	return nil
}

// ApiRequest is a wrapper for apiRequest, meant for more streamlined error logging.
func (c *Client) ApiRequest(ctx context.Context, method, url string, body io.Reader, target interface{}) error {
	if err := c.apiRequest(ctx, method, url, body, target); err != nil {
		slog.Warn("Connectwise API Error", "error", err)
		return fmt.Errorf("running ConnectWise PSA API request: %w", err)
	}

	return nil
}

func (c *Client) apiRequest(ctx context.Context, method, url string, body io.Reader, target interface{}) error {
	const maxRetries = 3
	var retryAfter int

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return fmt.Errorf("an error occured creating the request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("clientId", c.clientId)
		req.Header.Set("Authorization", c.encodedCreds)

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
					slog.Warn("failed to parse Retry-After header", "error", err)
					retryAfter = 1
				}

			} else {
				retryAfter = 1
			}

			slog.Warn("rate limit exceeded, retrying",
				"retryAfter", retryAfter,
				"totalRetries", fmt.Sprintf("%d/%d", attempt, maxRetries))
		} else {
			retryAfter = 15
			slog.Warn("connectwise API request failed - waiting 15 seconds if retries remain", "statusCode", res.StatusCode,
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

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}
