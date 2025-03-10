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
	slog.Debug("migration.NewClient called")
	httpClient := http.DefaultClient

	return &Client{
		ZendeskClient: zendesk.NewClient(zendeskCreds, httpClient),
		CwClient:      psa.NewClient(cwCreds, httpClient),
	}
}

func (c *Client) ConnectionTest(ctx context.Context) error {
	slog.Debug("migration.Client.ConnectionTest called")
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

func (c *Client) CheckZendeskPSAFields(ctx context.Context) error {
	slog.Debug("migration.Client.CheckZendeskPSAFields called")
	tf, err := c.ZendeskClient.GetTicketFieldByTitle(ctx, ticketFieldTitle)
	if err != nil {
		slog.Info("no psa_ticket field found in zendesk - creating")
		tf, err = c.ZendeskClient.PostTicketField(ctx, "integer", ticketFieldTitle, fieldDescription)
		if err != nil {
			slog.Error("creating psa ticket field", "error", err)
			return fmt.Errorf("creating psa ticket field: %w", err)
		}
	}
	slog.Info("psa ticket field", "id", tf.Id)

	uf, err := c.ZendeskClient.GetUserFieldByKey(ctx, contactFieldKey)
	if err != nil {
		slog.Info("no psa_contact field found in zendesk - creating")
		uf, err = c.ZendeskClient.PostUserField(ctx, "integer", contactFieldKey, contactFieldTitle, fieldDescription)
		if err != nil {
			slog.Error("creating psa contact field", "error", err)
			return fmt.Errorf("creating psa contact field: %w", err)
		}
	}
	slog.Info("psa contact field", "id", uf.Id)

	cf, err := c.ZendeskClient.GetOrgFieldByKey(ctx, companyFieldKey)
	if err != nil {
		slog.Info("no psa_company field found in zendesk - creating")
		cf, err = c.ZendeskClient.PostOrgField(ctx, "integer", companyFieldKey, companyFieldTitle, fieldDescription)
		if err != nil {
			slog.Error("creating psa company field", "error", err)
			return fmt.Errorf("creating psa company field: %w", err)
		}
	}
	slog.Info("psa company field", "id", cf.Id)

	fmt.Println("PSA Ticket Field ID:", tf.Id)
	fmt.Println("PSA Contact Field ID:", uf.Id)
	fmt.Println("PSA Contact Field ID:", cf.Id)

	return nil
}
