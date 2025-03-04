package migration

import "github.com/dsrosen/zendesk-connectwise-migrator/zendesk"

type Client struct {
	zendeskClient *zendesk.Client
}

func NewClient(zendeskClient *zendesk.Client) *Client {
	return &Client{
		zendeskClient: zendeskClient,
	}
}
