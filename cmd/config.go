package cmd

import "github.com/dsrosen/zendesk-connectwise-migrator/zendesk"

type config struct {
}

type zendeskConfig struct {
	ApiCreds zendesk.Creds
}
