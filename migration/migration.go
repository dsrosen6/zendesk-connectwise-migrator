package migration

import (
	"context"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/cw"
	"github.com/dsrosen/zendesk-connectwise-migrator/zendesk"
	"log/slog"
	"net/http"
)

type Client struct {
	ZendeskClient *zendesk.Client
	CwClient      *cw.Client
}

type Agent struct {
	Name      string `mapstructure:"name" json:"name"`
	ZendeskId int    `mapstructure:"zendesk_user_id" json:"zendesk_user_id"`
	CwId      int    `mapstructure:"connectwise_member_id" json:"connectwise_member_id"`
}

func NewClient(zendeskCreds zendesk.Creds, cwCreds cw.Creds) *Client {
	httpClient := http.DefaultClient

	return &Client{
		ZendeskClient: zendesk.NewClient(zendeskCreds, httpClient),
		CwClient:      cw.NewClient(cwCreds, httpClient),
	}
}

func (c *Client) ConnectionTest(ctx context.Context) error {
	testFailed := false
	if err := c.ZendeskClient.ConnectionTest(ctx); err != nil {
		slog.Error("testConnectionCmd", "action", "client.ZendeskClient.TestConnection", "error", err)
		fmt.Printf("Zendesk: Fail // %v\n", err)
		testFailed = true
	} else {
		fmt.Println("Zendesk: Success")
	}

	if err := c.CwClient.ConnectionTest(ctx); err != nil {
		slog.Error("testConnectionCmd", "action", "client.CwClient.TestConnection", "error", err)
		fmt.Printf("ConnectWise: Fail // %v\n", err)
		testFailed = true

	} else {
		fmt.Println("ConnectWise: Success")
	}

	if testFailed {
		fmt.Println("API Connection test failed - please review your config variables.")
		return nil
	}

	fmt.Println("API Connection test successful!")
	return nil
}
