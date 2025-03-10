package migration

import (
	"context"
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
	psaTicketFieldTitle  = "PSA Ticket"
	psaContactFieldTitle = "PSA Contact"
	psaContactFieldKey   = "psa_contact"
	psaCompanyFieldTitle = "PSA Company"
	psaCompanyFieldKey   = "psa_company"
	psaFieldDescription  = "Created by Zendesk to ConnectWise PSA Migration utility"
)

var (
	psaTicketFieldId  int64
	psaContactFieldId int64
	psaCompanyFieldId int64
)

type Client struct {
	ZendeskClient *zendesk.Client
	CwClient      *psa.Client
	Cfg           *Config
}

type Config struct {
	Zendesk ZdCfg `mapstructure:"zendesk" json:"zendesk"`
	CW      CwCfg `mapstructure:"connectwise_psa" json:"connectwise_psa"`
}

type ZdCfg struct {
	Creds         zendesk.Creds `mapstructure:"api_creds" json:"api_creds"`
	TagsToMigrate []string      `mapstructure:"tags_to_migrate" json:"tags_to_migrate"`
}

type CwCfg struct {
	Creds              psa.Creds `mapstructure:"api_creds" json:"api_creds"`
	ClosedStatusId     int       `mapstructure:"closed_status_id" json:"closed_status_id"`
	OpenStatusId       int       `mapstructure:"open_status_id" json:"open_status_id"`
	DestinationBoardId int       `mapstructure:"destination_board_id" json:"destination_board_id"`
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
	tf, err := c.ZendeskClient.GetTicketFieldByTitle(ctx, psaTicketFieldTitle)
	if err != nil {
		slog.Info("no psa_ticket field found in zendesk - creating")
		tf, err = c.ZendeskClient.PostTicketField(ctx, "integer", psaTicketFieldTitle, psaFieldDescription)
		if err != nil {
			slog.Error("creating psa ticket field", "error", err)
			return fmt.Errorf("creating psa ticket field: %w", err)
		}
	}

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
	psaTicketFieldId = tf.Id
	psaContactFieldId = uf.Id
	psaCompanyFieldId = cf.Id
	slog.Debug("CheckZendeskPSAFields", "ticketField", psaTicketFieldId, "userField", psaContactFieldId, "orgField", psaCompanyFieldId)
	return nil
}

func (c *Client) ZendeskTagForm(ctx context.Context) error {
	tags, err := c.ZendeskClient.GetTags(ctx)
	if err != nil {
		return fmt.Errorf("getting tags: %w", err)
	}

	var tagNames []string
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
	}

	var chosenTags []string
	input := huh.NewMultiSelect[string]().
		Title("Select Zendesk tags to migrate").
		Options(huh.NewOptions(tagNames...)...).
		Value(&chosenTags)

	if err := input.WithTheme(huh.ThemeBase16()).Run(); err != nil {
		return fmt.Errorf("running tag selection form: %w", err)
	}

	viper.Set("zendesk.tags_to_migrate", chosenTags)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
