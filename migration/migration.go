package migration

import (
	"github.com/dsrosen/zendesk-connectwise-migrator/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/zendesk"
)

type Client struct {
	zendeskClient *zendesk.Client
}

type Agent struct {
	Name      string `json:"name"`
	ZendeskId int    `json:"zendesk_user_id"`
	CwId      int    `json:"connectwise_member_id"`
}

func NewClient(zendeskCreds *zendesk.Creds, cwCreds *psa.Creds) *Client {
	return &Client{
		zendeskClient: zendeskClient,
	}
}
