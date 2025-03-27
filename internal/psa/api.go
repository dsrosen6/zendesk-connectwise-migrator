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
	"strings"
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

type PaginationDetails struct {
	HasMorePages bool
	NextLink     string
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

	if _, err := c.ApiRequest(ctx, "GET", url, nil, &co); err != nil {
		return errors.New("failed to connect to Connectwise API")
	}

	return nil
}

// ApiRequest is a wrapper for apiRequest, meant for more streamlined error logging.
func (c *Client) ApiRequest(ctx context.Context, method, url string, body io.Reader, target interface{}) (PaginationDetails, error) {
	pagination, err := c.apiRequest(ctx, method, url, body, target)
	if err != nil {
		return pagination, fmt.Errorf("running ConnectWise PSA API request: %w", err)
	}

	return pagination, nil
}

func (c *Client) apiRequest(ctx context.Context, method, url string, body io.Reader, target interface{}) (PaginationDetails, error) {
	const maxRetries = 3
	var retryAfter int
	p := &PaginationDetails{
		HasMorePages: false,
		NextLink:     "",
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return *p, fmt.Errorf("an error occured creating the request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("clientId", c.clientId)
		req.Header.Set("Authorization", c.encodedCreds)

		res, err := c.httpClient.Do(req)
		if err != nil {
			return *p, fmt.Errorf("an error occured sending the request: %w", err)
		}

		if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated {
			data, err := io.ReadAll(res.Body)
			if err != nil {
				return *p, fmt.Errorf("an error occured reading the response body: %w", err)
			}

			if target != nil {
				if err := json.Unmarshal(data, target); err != nil {
					return *p, fmt.Errorf("an error occured unmarshaling the response to JSON: %w", err)
				}
			}

			linkHeader := res.Header.Get("Link")
			if linkHeader != "" {
				if nextUrl, found := parseLinkHeader(linkHeader, "next"); found {
					p.HasMorePages = true
					p.NextLink = nextUrl
				}
			}

			return *p, nil
		}

		if res.StatusCode == http.StatusTooManyRequests {
			retryAfterHeader := res.Header.Get("Retry-After")

			if retryAfterHeader != "" {
				retryAfter, err = strconv.Atoi(retryAfterHeader)
				if err != nil {
					retryAfter = 1
				}

			} else {
				retryAfter = 1
			}

			slog.Debug("rate limit exceeded, retrying",
				"retryAfter", retryAfter,
				"totalRetries", fmt.Sprintf("%d/%d", attempt, maxRetries))
		} else {
			retryAfter = 5
			errorText, _ := io.ReadAll(res.Body)
			slog.Debug("connectwise API request failed - waiting 5 seconds if retries remain", "statusCode", res.StatusCode, "errorText", string(errorText),
				"totalRetries", fmt.Sprintf("%d/%d", attempt, maxRetries))
		}

		err = res.Body.Close()
		if err != nil {
			return *p, err
		}
		time.Sleep(time.Duration(retryAfter) * time.Second)
	}

	return *p, fmt.Errorf("max retries exceeded")
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func parseLinkHeader(linkHeader, rel string) (string, bool) {
	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		parts := strings.Split(strings.TrimSpace(link), ";")
		if len(parts) < 2 {
			continue
		}
		urlPart := strings.Trim(parts[0], "<>")
		relPart := strings.TrimSpace(parts[1])
		if relPart == fmt.Sprintf(`rel="%s"`, rel) {
			return urlPart, true
		}
	}

	return "", false
}
