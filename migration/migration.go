package migration

import (
	"github.com/dsrosen/zendesk-connectwise-migrator/cw"
	"github.com/dsrosen/zendesk-connectwise-migrator/zendesk"
	"net/http"
)

type Client struct {
	ZendeskClient *zendesk.Client
	CwClient      *cw.Client
}

type Agent struct {
	Name      string `mapstructure:"name"`
	ZendeskId int    `mapstructure:"zendeskUserId"`
	CwId      int    `mapstructure:"connectwiseMemberId"`
}

func NewClient(zendeskCreds zendesk.Creds, cwCreds cw.Creds) *Client {
	httpClient := http.DefaultClient

	return &Client{
		ZendeskClient: zendesk.NewClient(zendeskCreds, httpClient),
		CwClient:      cw.NewClient(cwCreds, httpClient),
	}
}
