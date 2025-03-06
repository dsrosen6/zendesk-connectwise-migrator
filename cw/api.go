package cw

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
	CompanyId  string `json:"companyId"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
	ClientId   string `json:"clientId"`
}

func NewClient(creds Creds, httpClient *http.Client) *Client {
	username := fmt.Sprintf("%s+%s", creds.CompanyId, creds.PublicKey)

	return &Client{
		encodedCreds: basicAuth(username, creds.PrivateKey),
		clientId:     creds.ClientId,
		httpClient:   httpClient,
	}
}

func (c *Client) apiRequest(ctx context.Context, method, url string, body io.Reader, target interface{}) error {
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

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}
