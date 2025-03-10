package psa

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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
	slog.Debug("psa.NewClient called")
	username := fmt.Sprintf("%s+%s", creds.CompanyId, creds.PublicKey)

	return &Client{
		encodedCreds: basicAuth(username, creds.PrivateKey),
		clientId:     creds.ClientId,
		httpClient:   httpClient,
	}
}

func (c *Client) ConnectionTest(ctx context.Context) error {
	slog.Debug("psa.ConnectionTest called")
	url := fmt.Sprintf("%s/company/companies?pageSize=1", baseUrl)
	co := Companies{}

	if err := c.apiRequest(ctx, "GET", url, nil, &co); err != nil {
		return err
	}

	return nil
}

func (c *Client) apiRequest(ctx context.Context, method, url string, body io.Reader, target interface{}) error {
	slog.Debug("psa.GetCompanyByName called")
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("clientId", c.clientId)
	req.Header.Set("Authorization", c.encodedCreds)

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

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}
