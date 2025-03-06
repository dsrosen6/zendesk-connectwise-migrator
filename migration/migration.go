package migration

import (
	"github.com/dsrosen/zendesk-connectwise-migrator/cw"
	"github.com/dsrosen/zendesk-connectwise-migrator/zendesk"
	"net/http"
)

type Client struct {
	zendeskClient *zendesk.Client
	cwClient      *cw.Client
}

type Agent struct {
	Name      string `json:"name"`
	ZendeskId int    `json:"zendeskUserId"`
	CwId      int    `json:"connectwiseMemberId"`
}

func NewClient(zendeskCreds zendesk.Creds, cwCreds cw.Creds) *Client {
	httpClient := http.DefaultClient

	return &Client{
		zendeskClient: zendesk.NewClient(zendeskCreds, httpClient),
		cwClient:      cw.NewClient(cwCreds, httpClient),
	}
}
