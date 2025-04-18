package migration

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"github.com/spf13/viper"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func Run(opts CliOptions) error {
	ctx := context.Background()
	dir, err := makeMigrationDir()
	if err != nil {
		return fmt.Errorf("creating migration directory: %w", err)
	}

	logFile, err := openLogFile(filepath.Join(dir, "migration.log"))
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	if err := setLogger(logFile, opts.Debug); err != nil {
		return fmt.Errorf("setting logger: %w", err)
	}

	client, err := runStartup(ctx, dir, opts)
	if err != nil {
		slog.Error("running startup", "error", err)
		return err
	}

	model, err := newModel(ctx, client)
	if err != nil {
		return fmt.Errorf("initializing terminal interface: %w", err)
	}

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func makeMigrationDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting user home directory: %w", err)
	}

	migrationDir := filepath.Join(home, "ticket-migration")
	if err := os.MkdirAll(migrationDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("creating migration directory: %w", err)
	}

	return migrationDir, nil
}

func runStartup(ctx context.Context, dir string, opts CliOptions) (*Client, error) {
	cfg, err := InitConfig(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config: %w", err)
	}

	cfg.CliOptions = opts
	viper.Set("ticket_limit", opts.TicketLimit)
	viper.Set("migrate_open_tickets", opts.MigrateOpenTickets)
	viper.Set("output_levels", opts.OutputLevels)

	slog.Info("startup options", "opts", opts)

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
	httpClient := &http.Client{
		Transport: newTransport(),
	}
	
	return &Client{
		ZendeskClient: zendesk.NewClient(zendeskCreds, httpClient),
		CwClient:      psa.NewClient(cwCreds, httpClient),
		Cfg:           cfg,
	}
}

func newTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:        100,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
		MaxIdleConnsPerHost: 100,
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

func convertStrTime(date string) (time.Time, error) {
	layout := "2006-01-02"
	d, err := time.Parse(layout, date)
	if err != nil {
		return time.Time{}, fmt.Errorf("converting time string to datetime format: %w", err)
	}

	return d, nil
}
