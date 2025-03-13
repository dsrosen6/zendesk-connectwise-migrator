package migration

import (
	"context"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/apis/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/apis/zendesk"
	"log/slog"
	"net/http"
	"strings"
)

const (
	psaContactFieldTitle = "PSA Contact"
	psaContactFieldKey   = "psa_contact"
	psaCompanyFieldTitle = "PSA Company"
	psaCompanyFieldKey   = "psa_company"
	psaFieldDescription  = "Created by Zendesk to ConnectWise PSA Migration utility"
)

type Client struct {
	ZendeskClient *zendesk.Client
	CwClient      *psa.Client
	Cfg           *Config
}

type Agent struct {
	Name      string `mapstructure:"name" json:"name"`
	ZendeskId int    `mapstructure:"zendesk_user_id" json:"zendesk_user_id"`
	CwId      int    `mapstructure:"connectwise_member_id" json:"connectwise_member_id"`
}

func RunStartup(ctx context.Context) (*Client, error) {
	cfg, err := InitConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config: %w", err)
	}

	if err := cfg.ValidateAndPrompt(); err != nil {
		return nil, fmt.Errorf("validating and prompt config: %w", err)
	}
	slog.Info("Config Validated")

	client := newClient(cfg.Zendesk.Creds, cfg.CW.Creds, cfg)

	if err := client.testConnection(ctx); err != nil {
		slog.Error("ConnectionTest: error", "error", err)
	}

	if err := client.Cfg.validateZendeskCustomFields(); err != nil {
		if err := client.processZendeskPsaForms(ctx); err != nil {
			return nil, fmt.Errorf("getting zendesk fields: %w", err)
		}
	}

	if err := client.Cfg.validateConnectwiseBoardId(); err != nil {
		if err := client.runBoardForm(ctx); err != nil {
			return nil, fmt.Errorf("running board form: %w", err)
		}
	}

	if err := client.Cfg.validateConnectwiseStatuses(); err != nil {
		if err := client.runBoardStatusForm(ctx, cfg.CW.DestinationBoardId); err != nil {
			return nil, fmt.Errorf("running board status form: %w", err)
		}
	}

	return client, nil
}

func newClient(zendeskCreds zendesk.Creds, cwCreds psa.Creds, cfg *Config) *Client {
	httpClient := http.DefaultClient

	return &Client{
		ZendeskClient: zendesk.NewClient(zendeskCreds, httpClient),
		CwClient:      psa.NewClient(cwCreds, httpClient),
		Cfg:           cfg,
	}
}

func (c *Client) testConnection(ctx context.Context) error {
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

	slog.Info("ConnectionTest: success")
	return nil
}
