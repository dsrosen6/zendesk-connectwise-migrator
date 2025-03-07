package cmd

import (
	"errors"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/cw"
	"github.com/dsrosen/zendesk-connectwise-migrator/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/zendesk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"strings"
)

const (
	configFileSubPath = "/migrator_config.json"
)

var (
	cfgFile string
	config  Config
)

type Config struct {
	AgentMappings []migration.Agent `mapstructure:"agent_mappings" json:"agent_mappings"`
	Zendesk       ZendeskConfig     `mapstructure:"zendesk" json:"zendesk"`
	CW            CwConfig          `mapstructure:"connectwise_psa" json:"connectwise_psa"`
}

type ZendeskConfig struct {
	ApiCreds zendesk.Creds `mapstructure:"api_creds" json:"api_creds"`
	FieldIds FieldIds      `mapstructure:"field_ids" json:"field_ids"`
}

type FieldIds struct {
	PSACompanyId string `mapstructure:"psa_company_id" json:"psa_company_id"`
	PSAContactId string `mapstructure:"psa_contact_id" json:"psa_contact_id"`
	PSATicketId  int    `mapstructure:"psa_ticket_id" json:"psa_ticket_id"`
}

type CwConfig struct {
	ApiCreds           cw.Creds `mapstructure:"api_creds" json:"api_creds"`
	ClosedStatusId     int      `mapstructure:"closed_status_id" json:"closed_status_id"`
	OpenStatusId       int      `mapstructure:"open_status_id" json:"open_status_id"`
	DestinationBoardId int      `mapstructure:"destination_board_id" json:"destination_board_id"`
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
				fmt.Println("Error creating default config file:", err)
				os.Exit(1)
			}
			fmt.Println("Config file created - location:", path)
			fmt.Println("Please fill in the necessary fields and run the program again.")
		} else {
			fmt.Println("Error reading config file:", err)
			os.Exit(1)
		}
	}
}

func setCfgDefaults() {
	slog.Debug("setting config defaults")
	viper.SetDefault("agentMappings", []migration.Agent{{}}) // prefill with empty agent
	viper.SetDefault("zendesk", ZendeskConfig{})
	viper.SetDefault("connectwise_psa", CwConfig{})
}

func validateConfig(cfg Config) error {
	slog.Debug("validating config")
	if err := validateRequiredValues(cfg); err != nil {
		return err
	}

	if err := validateAgentMappings(cfg.AgentMappings); err != nil {
		return err
	}

	return nil
}

func validateRequiredValues(cfg Config) error {
	slog.Debug("validating required fields")
	var missing []string

	keysWithStrVal := []string{
		cfg.Zendesk.ApiCreds.Token,
		cfg.Zendesk.ApiCreds.Username,
		cfg.Zendesk.ApiCreds.Subdomain,
		cfg.CW.ApiCreds.CompanyId,
		cfg.CW.ApiCreds.PublicKey,
		cfg.CW.ApiCreds.PrivateKey,
		cfg.CW.ApiCreds.ClientId,
		cfg.Zendesk.FieldIds.PSACompanyId,
		cfg.Zendesk.FieldIds.PSAContactId,
	}

	keysWithIntVal := []int{
		cfg.CW.ClosedStatusId,
		cfg.CW.OpenStatusId,
		cfg.CW.DestinationBoardId,
		cfg.Zendesk.FieldIds.PSATicketId,
	}

	// if value is empty, add key to missing
	for _, key := range keysWithStrVal {
		if key == "" {
			slog.Warn("missing required config value", "key", key)
			missing = append(missing, key)
		}
	}

	// if value is 0, add key to missing
	for _, key := range keysWithIntVal {
		if key == 0 {
			slog.Warn("missing required config value", "key", key)
			missing = append(missing, fmt.Sprintf("%d", key))
		}
	}

	if len(missing) > 0 {
		slog.Error("missing required config values", "missing", missing)
		return errors.New("missing 1 or more required config values")
	}

	return nil
}

func validateAgentMappings(agents []migration.Agent) error {
	slog.Debug("validating agent mappings")
	missingAgent := false
	for _, agent := range agents {
		var missingAgentVals []string
		if agent.Name == "" {
			missingAgentVals = append(missingAgentVals, "name")
		}

		if agent.ZendeskId == 0 {
			missingAgentVals = append(missingAgentVals, "zendeskUserId")
		}

		if agent.CwId == 0 {
			missingAgentVals = append(missingAgentVals, "connectwiseMemberId")
		}

		if len(missingAgentVals) > 0 {
			slog.Warn("agent mapping is missing required fields", "missing", missingAgentVals)
			fmt.Printf("\nAn agent mapping is missing: %s\n   current values:\n       name: %s\n       zendeskUserId: %d\n       connectwiseMemberId: %d\n",
				strings.Join(missingAgentVals, ","), agent.Name, agent.ZendeskId, agent.CwId)

			missingAgent = true
		}
	}

	if missingAgent {
		slog.Error("agent mapping is missing required fields")
		return errors.New("missing agent mapping")
	}

	return nil
}
