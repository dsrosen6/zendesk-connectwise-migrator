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

type RateLimitErr struct{}

func (r RateLimitErr) Error() string {
	return "rate limit exceeded"
}

type BadGatewayErr struct{}

func (b BadGatewayErr) Error() string {
	return "bad gateway"
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
	slog.Debug("psa.apiRequest: called", "method", method, "url", url)
	const maxRetries = 3
	var retryAfter int
	p := &PaginationDetails{
		HasMorePages: false,
		NextLink:     "",
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			slog.Debug("psa.apiRequest: making additional attempt", "method", method, "url", url, "attempt", attempt)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			slog.Debug("psa.apiRequest: error creating request", "method", method, "url", url, "error", err)
			return *p, fmt.Errorf("an error occured creating the request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("clientId", c.clientId)
		req.Header.Set("Authorization", c.encodedCreds)

		res, err := c.httpClient.Do(req)
		if err != nil {
			slog.Debug("psa.apiRequest: error sending request", "method", method, "url", url, "error", err)
			return *p, fmt.Errorf("an error occured sending the request: %w", err)
		}

		err = func() error {
			defer res.Body.Close()
			if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated {
				data, err := io.ReadAll(res.Body)
				if err != nil {
					slog.Debug("psa.apiRequest: error reading response body", "method", method, "url", url, "error", err)
					return fmt.Errorf("an error occured reading the response body: %w", err)
				}

				if target != nil {
					if err := json.Unmarshal(data, target); err != nil {
						slog.Debug("psa.apiRequest: error unmarshaling response", "data", string(data), "error", err)
						return fmt.Errorf("an error occured unmarshaling the response to JSON: %w", err)
					}
				}

				linkHeader := res.Header.Get("Link")
				if linkHeader != "" {
					if nextUrl, found := parseLinkHeader(linkHeader, "next"); found {
						p.HasMorePages = true
						p.NextLink = nextUrl
					}
				}

				return nil
			}

			if res.StatusCode == http.StatusTooManyRequests {
				retryAfterHeader := res.Header.Get("Retry-After")

				if retryAfterHeader != "" {
					retryAfter, err = strconv.Atoi(retryAfterHeader)
					if err != nil {
						slog.Debug("psa.apiRequest: error parsing Retry-After header", "headerValue", retryAfterHeader, "error", err)
						retryAfter = 1
					}

				} else {
					slog.Debug("psa.apiRequest: no Retry-After header provided, defaulting to 1 second")
					retryAfter = 1
				}

				slog.Debug("psa.apiRequest: rate limit exceeded, retrying after", "retryAfter", retryAfter, "attempt", attempt)
				return RateLimitErr{}
			}

			if res.StatusCode == http.StatusBadGateway || res.StatusCode == http.StatusServiceUnavailable || res.StatusCode == http.StatusInternalServerError || res.StatusCode == http.StatusGatewayTimeout {
				retryAfter = 10
				return BadGatewayErr{}
			}

			retryAfter = 10
			errorText, _ := io.ReadAll(res.Body)
			slog.Debug("psa.apiRequest: response status", "statusCode", res.StatusCode, "responseBody", string(errorText))
			return fmt.Errorf("received non-200 response: %s (status code: %d)", res.Status, res.StatusCode)
		}()

		if err == nil {
			slog.Debug("psa.apiRequest: request successful", "method", method, "url", url)
			return *p, nil
		}

		if !errors.As(err, &RateLimitErr{}) && !errors.As(err, &BadGatewayErr{}) {
			slog.Debug("psa.apiRequest: non-rate limit or gateway error encountered", "error", err)
			return *p, err
		}

		time.Sleep(time.Duration(retryAfter) * time.Second)
	}

	slog.Debug("psa.apiRequest: max retries reached", "method", method, "url", url, "maxRetries", maxRetries)
	return PaginationDetails{}, fmt.Errorf("max retries exceeded for API request: %s %s", method, url)
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
