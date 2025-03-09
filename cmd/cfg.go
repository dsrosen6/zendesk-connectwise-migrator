package cmd

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/dsrosen/zendesk-connectwise-migrator/cw"
	"github.com/dsrosen/zendesk-connectwise-migrator/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/zendesk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

const (
	configFileSubPath = "/migrator_config.json"
)

var (
	cfgFile  string
	conf     cfg
	testConn = "Y"
)

type cfg struct {
	//AgentMappings []migration.Agent `mapstructure:"agent_mappings" json:"agent_mappings"`
	Zendesk zdCfg `mapstructure:"zendesk" json:"zendesk"`
	CW      cwCfg `mapstructure:"connectwise_psa" json:"connectwise_psa"`
}

type zdCfg struct {
	Creds    zendesk.Creds `mapstructure:"api_creds" json:"api_creds"`
	FieldIds zdFieldIds    `mapstructure:"field_ids" json:"field_ids"`
}

type zdFieldIds struct {
	PSACompanyId int `mapstructure:"psa_company_id" json:"psa_company_id"`
	PSAContactId int `mapstructure:"psa_contact_id" json:"psa_contact_id"`
	PSATicketId  int `mapstructure:"psa_ticket_id" json:"psa_ticket_id"`
}

type cwCfg struct {
	Creds              cw.Creds `mapstructure:"api_creds" json:"api_creds"`
	ClosedStatusId     int      `mapstructure:"closed_status_id" json:"closed_status_id"`
	OpenStatusId       int      `mapstructure:"open_status_id" json:"open_status_id"`
	DestinationBoardId int      `mapstructure:"destination_board_id" json:"destination_board_id"`
}

var configCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"cfg"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := conf.credsForm().Run(); err != nil {
			return err
		}

		viper.Set("zendesk", conf.Zendesk)
		viper.Set("connectwise_psa", conf.CW)

		if err := viper.WriteConfig(); err != nil {
			return err
		}

		client = migration.NewClient(conf.Zendesk.Creds, conf.CW.Creds)
		if strings.ToLower(testConn) == "y" {
			return client.ConnectionTest(ctx)
		}

		return nil
	},
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
		} else {
			fmt.Println("Error reading config file:", err)
			os.Exit(1)
		}
	}
}

func (cfg *cfg) validateConfig() error {
	slog.Debug("validating config")
	if err := cfg.validateRequiredValues(); err != nil {
		return err
	}

	//if err := validateAgentMappings(cfg.AgentMappings); err != nil {
	//	return err
	//}

	return nil
}

func (cfg *cfg) validateRequiredValues() error {
	slog.Debug("validating required fields")
	var missing []string

	keysWithStrVal := []string{
		cfg.Zendesk.Creds.Token,
		cfg.Zendesk.Creds.Username,
		cfg.Zendesk.Creds.Subdomain,
		cfg.CW.Creds.CompanyId,
		cfg.CW.Creds.PublicKey,
		cfg.CW.Creds.PrivateKey,
		cfg.CW.Creds.ClientId,
	}

	keysWithIntVal := []int{
		cfg.CW.ClosedStatusId,
		cfg.CW.OpenStatusId,
		cfg.CW.DestinationBoardId,
		cfg.Zendesk.FieldIds.PSACompanyId,
		cfg.Zendesk.FieldIds.PSAContactId,
		cfg.Zendesk.FieldIds.PSATicketId,
	}

	for _, key := range keysWithStrVal {
		if key == "" {
			slog.Warn("missing required config value", "key", key)
			missing = append(missing, key)
		}
	}

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

func (cfg *cfg) credsForm() *huh.Form {
	return huh.NewForm(
		inputGroup("Zendesk Token", cfg.Zendesk.Creds.Token, requiredInput, true),
		inputGroup("Zendesk Username", cfg.Zendesk.Creds.Username, requiredInput, true),
		inputGroup("Zendesk Subdomain", cfg.Zendesk.Creds.Subdomain, requiredInput, true),
		inputGroup("ConnectWise Company ID", cfg.CW.Creds.CompanyId, requiredInput, true),
		inputGroup("ConnectWise Public Key", cfg.CW.Creds.PublicKey, requiredInput, true),
		inputGroup("ConnectWise Private Key", cfg.CW.Creds.PrivateKey, requiredInput, true),
		inputGroup("ConnectWise Client ID", cfg.CW.Creds.ClientId, requiredInput, true),
		inputGroup("Test connection? (Y/N)", testConn, requiredInput, true),
	).WithHeight(3).WithShowHelp(false).WithTheme(huh.ThemeBase16())
}

// inputGroup creates a huh Group with an input field, this is just to make cfg.credsForm prettier.
func inputGroup(title string, value string, validate func(string) error, inline bool) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title(title).
			Placeholder(value).
			Validate(validate).
			Inline(inline).
			Value(&value),
	)
}

func strToInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
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
	//viper.SetDefault("agentMappings", []migration.Agent{{}}) // prefill with empty agent
	viper.SetDefault("zendesk", zdCfg{})
	viper.SetDefault("connectwise_psa", cwCfg{})
}

//func validateAgentMappings(agents []migration.Agent) error {
//	slog.Debug("validating agent mappings")
//	missingAgent := false
//	for _, agent := range agents {
//		var missingAgentVals []string
//		if agent.Name == "" {
//			missingAgentVals = append(missingAgentVals, "name")
//		}
//
//		if agent.ZendeskId == 0 {
//			missingAgentVals = append(missingAgentVals, "zendeskUserId")
//		}
//
//		if agent.CwId == 0 {
//			missingAgentVals = append(missingAgentVals, "connectwiseMemberId")
//		}
//
//		if len(missingAgentVals) > 0 {
//			slog.Warn("agent mapping is missing required fields", "missing", missingAgentVals)
//			fmt.Printf("\nAn agent mapping is missing: %s\n   current values:\n       name: %s\n       zendeskUserId: %d\n       connectwiseMemberId: %d\n",
//				strings.Join(missingAgentVals, ","), agent.Name, agent.ZendeskId, agent.CwId)
//
//			missingAgent = true
//		}
//	}
//
//	if missingAgent {
//		slog.Error("agent mapping is missing required fields")
//		return errors.New("missing agent mapping")
//	}
//
//	return nil
//}

func init() {
	rootCmd.AddCommand(configCmd)
}
