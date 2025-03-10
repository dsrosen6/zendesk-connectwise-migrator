package migration

import (
	"context"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"net/http"
	"strings"
)

const (
	ticketFieldTitle  = "PSA Ticket"
	contactFieldTitle = "PSA Contact"
	contactFieldKey   = "psa_contact"
	companyFieldTitle = "PSA Company"
	companyFieldKey   = "psa_company"
	fieldDescription  = "Created by Zendesk to ConnectWise PSA Migration utility"
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
	var failedTests []string
	if err := c.ZendeskClient.ConnectionTest(ctx); err != nil {
		failedTests = append(failedTests, "zendesk")
	}

	if err := c.CwClient.ConnectionTest(ctx); err != nil {
		failedTests = append(failedTests, "connectwise")
	}

	if len(failedTests) > 0 {
		slog.Error("ConnectionTest: error", "failedTests", failedTests)
		return fmt.Errorf("failed connection tests: %v", strings.Join(failedTests, ","))
	}

	return nil
}

func (c *Client) CheckZendeskPSAFields(ctx context.Context) error {
	tf, err := c.ZendeskClient.GetTicketFieldByTitle(ctx, ticketFieldTitle)
	if err != nil {
		slog.Info("no psa_ticket field found in zendesk - creating")
		tf, err = c.ZendeskClient.PostTicketField(ctx, "integer", ticketFieldTitle, fieldDescription)
		if err != nil {
			slog.Error("creating psa ticket field", "error", err)
			return fmt.Errorf("creating psa ticket field: %w", err)
		}
	}

	uf, err := c.ZendeskClient.GetUserFieldByKey(ctx, contactFieldKey)
	if err != nil {
		slog.Info("no psa_contact field found in zendesk - creating")
		uf, err = c.ZendeskClient.PostUserField(ctx, "integer", contactFieldKey, contactFieldTitle, fieldDescription)
		if err != nil {
			slog.Error("creating psa contact field", "error", err)
			return fmt.Errorf("creating psa contact field: %w", err)
		}
	}

	cf, err := c.ZendeskClient.GetOrgFieldByKey(ctx, companyFieldKey)
	if err != nil {
		slog.Info("no psa_company field found in zendesk - creating")
		cf, err = c.ZendeskClient.PostOrgField(ctx, "integer", companyFieldKey, companyFieldTitle, fieldDescription)
		if err != nil {
			slog.Error("creating psa company field", "error", err)
			return fmt.Errorf("creating psa company field: %w", err)
		}
	}
	slog.Debug("CheckZendeskPSAFields", "ticketField", tf, "userField", uf, "orgField", cf)
	return nil
}
