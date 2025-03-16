package migration

import (
	"context"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"github.com/spf13/viper"
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

	if err := cfg.validatePreClient(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	slog.Info("config validated")

	client := newClient(cfg.Zendesk.Creds, cfg.Connectwise.Creds, cfg)

	if err := client.validatePostClient(ctx); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	slog.Debug("config details", "config", cfg)
	if err := viper.WriteConfig(); err != nil {
		return nil, fmt.Errorf("writing config file: %w", err)
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
		slog.Error("zendesk api connection test", "error", err)
		failedTests = append(failedTests, "zendesk")
	}

	if err := c.CwClient.ConnectionTest(ctx); err != nil {
		slog.Error("connectwise api connection test", "error", err)
		failedTests = append(failedTests, "connectwise")
	}

	if len(failedTests) > 0 {
		slog.Error("connection test", "failedTests", failedTests)
		return fmt.Errorf("failed connection tests: %v", strings.Join(failedTests, ","))
	}

	slog.Info("connection tests successful")
	return nil
}
