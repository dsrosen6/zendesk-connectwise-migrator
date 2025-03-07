package migration

import (
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
	slog.Debug("creating migration client")
	httpClient := http.DefaultClient

	return &Client{
		ZendeskClient: zendesk.NewClient(zendeskCreds, httpClient),
		CwClient:      cw.NewClient(cwCreds, httpClient),
	}
}
