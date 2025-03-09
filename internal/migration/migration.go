package migration

import (
	"context"
	"errors"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"net/http"
)

type Client struct {
	ZendeskClient *zendesk.Client
	CwClient      *psa.Client
}

type Agent struct {
	Name      string `mapstructure:"name" json:"name"`
	ZendeskId int    `mapstructure:"zendesk_user_id" json:"zendesk_user_id"`
	CwId      int    `mapstructure:"connectwise_member_id" json:"connectwise_member_id"`
}

func NewClient(zendeskCreds zendesk.Creds, cwCreds psa.Creds) *Client {
	httpClient := http.DefaultClient

	return &Client{
		ZendeskClient: zendesk.NewClient(zendeskCreds, httpClient),
		CwClient:      psa.NewClient(cwCreds, httpClient),
	}
}

func (c *Client) ConnectionTest(ctx context.Context) error {
	testFailed := false
	if err := c.ZendeskClient.ConnectionTest(ctx); err != nil {
		slog.Error("testConnectionCmd", "action", "client.ZendeskClient.TestConnection", "error", err)
		fmt.Printf("✗ Zendesk // %v\n", err)
		testFailed = true
	} else {
		fmt.Println("✓ Zendesk")
	}

	if err := c.CwClient.ConnectionTest(ctx); err != nil {
		slog.Error("testConnectionCmd", "action", "client.CwClient.TestConnection", "error", err)
		fmt.Printf("✗ ConnectWise // %v\n", err)
		testFailed = true

	} else {
		fmt.Println("✓ ConnectWise")
	}

	if testFailed {
		slog.Error("connection test failed")
		return errors.New("one or more API connections failed")
	}

	fmt.Println("Connection test successful!")
	return nil
}
