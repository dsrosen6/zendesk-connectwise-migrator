package migration

import (
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/huh"
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

func Run(ctx context.Context) error {
	cfg, err := InitConfig()
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	if err := cfg.ValidateAndPrompt(); err != nil {
		return fmt.Errorf("validating and prompt config: %w", err)
	}
	slog.Info("Config Validated")

	client := NewClient(cfg.Zendesk.Creds, cfg.CW.Creds, cfg)

	if err := client.ConnectionTest(ctx); err != nil {
		return fmt.Errorf("connection test: %w", err)
	}

	if err := client.ValidateZendeskCustomFields(); err != nil {
		if err := client.GetOrCreateZendeskPsaFields(ctx); err != nil {
			return fmt.Errorf("getting zendesk fields: %w", err)
		}
	}

	if err := client.ValidateConnectwiseBoardId(ctx); err != nil {
		if err := client.RunBoardForm(ctx); err != nil {
			return fmt.Errorf("running board form: %w", err)
		}
	}
	
	return nil
}

func NewClient(zendeskCreds zendesk.Creds, cwCreds psa.Creds, cfg *Config) *Client {
	httpClient := http.DefaultClient

	return &Client{
		ZendeskClient: zendesk.NewClient(zendeskCreds, httpClient),
		CwClient:      psa.NewClient(cwCreds, httpClient),
		Cfg:           cfg,
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

	slog.Info("ConnectionTest: success")
	return nil
}
func (c *Client) ValidateZendeskCustomFields() error {
	if c.Cfg.Zendesk.FieldIds.PsaCompanyId == 0 || c.Cfg.Zendesk.FieldIds.PsaContactId == 0 {
		slog.Warn("no Zendesk custom field IDs set")
		return errors.New("no Zendesk custom field IDs set")
	}

	return nil
}

func (c *Client) GetOrCreateZendeskPsaFields(ctx context.Context) error {
	uf, err := c.ZendeskClient.GetUserFieldByKey(ctx, psaContactFieldKey)
	if err != nil {
		slog.Info("no psa_contact field found in zendesk - creating")
		uf, err = c.ZendeskClient.PostUserField(ctx, "integer", psaContactFieldKey, psaContactFieldTitle, psaFieldDescription)
		if err != nil {
			slog.Error("creating psa contact field", "error", err)
			return fmt.Errorf("creating psa contact field: %w", err)
		}
	}

	cf, err := c.ZendeskClient.GetOrgFieldByKey(ctx, psaCompanyFieldKey)
	if err != nil {
		slog.Info("no psa_company field found in zendesk - creating")
		cf, err = c.ZendeskClient.PostOrgField(ctx, "integer", psaCompanyFieldKey, psaCompanyFieldTitle, psaFieldDescription)
		if err != nil {
			slog.Error("creating psa company field", "error", err)
			return fmt.Errorf("creating psa company field: %w", err)
		}
	}

	c.Cfg.Zendesk.FieldIds.PsaContactId = uf.Id
	c.Cfg.Zendesk.FieldIds.PsaCompanyId = cf.Id
	viper.Set("zendesk.field_ids.psa_contact_id", uf.Id)
	viper.Set("zendesk.field_ids.psa_company_id", cf.Id)
	if err := viper.WriteConfig(); err != nil {
		slog.Error("writing zendesk psa fields to config", "error", err)
		return fmt.Errorf("writing zendesk psa fields to config: %w", err)
	}
	slog.Debug("CheckZendeskPSAFields", "userField", c.Cfg.Zendesk.FieldIds.PsaContactId, "orgField", c.Cfg.Zendesk.FieldIds.PsaCompanyId)
	return nil
}

func (c *Client) ValidateConnectwiseBoardId(ctx context.Context) error {
	if c.Cfg.CW.DestinationBoardId == 0 {
		slog.Warn("no destination board ID set")
		return errors.New("no destination board ID set")
	}

	return nil
}

func (c *Client) RunBoardForm(ctx context.Context) error {
	boards, err := c.CwClient.GetBoards(ctx)
	if err != nil {
		return fmt.Errorf("getting boards: %w", err)
	}

	var boardNames []string
	for _, board := range boards {
		boardNames = append(boardNames, board.Name)
	}

	boardsMap := make(map[string]int)
	for _, b := range boards {
		boardsMap[b.Name] = b.Id
	}

	var s string
	input := huh.NewSelect[string]().
		Title("Choose destination ConnectWise PSA board").
		Options(huh.NewOptions(boardNames...)...).
		Value(&s).
		WithTheme(huh.ThemeBase())

	if err := input.Run(); err != nil {
		return fmt.Errorf("running board form: %w", err)
	}

	if _, ok := boardsMap[s]; !ok {
		return errors.New("invalid board selection")
	}

	c.Cfg.CW.DestinationBoardId = boardsMap[s]
	viper.Set("connectwise_psa.destination_board_id", boardsMap[s])
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
