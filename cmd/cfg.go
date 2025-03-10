package cmd

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"strconv"
)

const (
	configFileSubPath = "/migrator_config.json"
)

var (
	cfgFile string
	conf    cfg
)

type cfg struct {
	Zendesk zdCfg `mapstructure:"zendesk" json:"zendesk"`
	CW      cwCfg `mapstructure:"connectwise_psa" json:"connectwise_psa"`
}

type zdCfg struct {
	Creds         zendesk.Creds `mapstructure:"api_creds" json:"api_creds"`
	TagsToMigrate []string      `mapstructure:"tags_to_migrate" json:"tags_to_migrate"`
}

type cwCfg struct {
	Creds              psa.Creds `mapstructure:"api_creds" json:"api_creds"`
	ClosedStatusId     int       `mapstructure:"closed_status_id" json:"closed_status_id"`
	OpenStatusId       int       `mapstructure:"open_status_id" json:"open_status_id"`
	DestinationBoardId int       `mapstructure:"destination_board_id" json:"destination_board_id"`
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Find home directory.
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in home directory with name ".zendesk-connectwise-migrator" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("json")
		viper.SetConfigName("migrator_config")
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			setCfgDefaults()
			path := home + configFileSubPath
			fmt.Println("Creating default config file")
			if err := viper.WriteConfigAs(path); err != nil {
				slog.Error("error creating default config file", "error", err)
				fmt.Println("Error creating default config file:", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("Error reading config file:", err)
			os.Exit(1)
		}
	}
}

func (cfg *cfg) validateConfig() error {
	slog.Debug("validating required fields")
	var missing []string

	requiredFields := map[string]string{
		"zendesk.api_creds.token":               cfg.Zendesk.Creds.Token,
		"zendesk.api_creds.username":            cfg.Zendesk.Creds.Username,
		"zendesk.api_creds.subdomain":           cfg.Zendesk.Creds.Subdomain,
		"connectwise_psa.api_creds.company_id":  cfg.CW.Creds.CompanyId,
		"connectwise_psa.api_creds.public_key":  cfg.CW.Creds.PublicKey,
		"connectwise_psa.api_creds.private_key": cfg.CW.Creds.PrivateKey,
		"connectwise_psa.api_creds.client_id":   cfg.CW.Creds.ClientId,
	}

	for k, v := range requiredFields {
		if v == "" {
			slog.Warn("missing required config value", "key", k)
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		slog.Error("missing required config values", "missing", missing)
		return errors.New("missing 1 or more required config values")
	}

	return nil
}

func (cfg *cfg) runCredsForm() error {
	if err := conf.credsForm().Run(); err != nil {
		slog.Error("error running creds form", "error", err)
		return fmt.Errorf("running creds form: %w", err)
	}
	slog.Debug("creds form completed", "cfg", cfg)

	viper.Set("zendesk", cfg.Zendesk)
	viper.Set("connectwise_psa", cfg.CW)
	if err := viper.WriteConfig(); err != nil {
		slog.Error("error writing config file", "error", err)
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func (cfg *cfg) credsForm() *huh.Form {
	return huh.NewForm(
		inputGroup("Zendesk Token", &cfg.Zendesk.Creds.Token, requiredInput, true),
		inputGroup("Zendesk Username", &cfg.Zendesk.Creds.Username, requiredInput, true),
		inputGroup("Zendesk Subdomain", &cfg.Zendesk.Creds.Subdomain, requiredInput, true),
		inputGroup("ConnectWise Company ID", &cfg.CW.Creds.CompanyId, requiredInput, true),
		inputGroup("ConnectWise Public Key", &cfg.CW.Creds.PublicKey, requiredInput, true),
		inputGroup("ConnectWise Private Key", &cfg.CW.Creds.PrivateKey, requiredInput, true),
		inputGroup("ConnectWise Client ID", &cfg.CW.Creds.ClientId, requiredInput, true),
	).WithHeight(3).WithShowHelp(false).WithTheme(huh.ThemeBase16())
}

// inputGroup creates a huh Group with an input field, this is just to make cfg.credsForm prettier.
func inputGroup(title string, value *string, validate func(string) error, inline bool) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title(title).
			Placeholder(*value).
			Validate(validate).
			Inline(inline).
			Value(value),
	)
}

func strToInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		slog.Error("error converting string to int", "error", err)
		return 0
	}

	return i
}

// Validator for required huh Input fields
func requiredInput(s string) error {
	if s == "" {
		return errors.New("field is required")
	}
	return nil
}

func setCfgDefaults() {
	slog.Debug("setting config defaults")
	viper.SetDefault("zendesk", zdCfg{})
	viper.SetDefault("connectwise_psa", cwCfg{})
}
