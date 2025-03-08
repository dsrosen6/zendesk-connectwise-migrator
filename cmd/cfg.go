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
)

const (
	configFileSubPath = "/migrator_config.json"
)

var (
	cfgFile  string
	conf     cfg
	testConn bool
)

type cfg struct {
	//AgentMappings []migration.Agent `mapstructure:"agent_mappings" json:"agent_mappings"`
	Zendesk zendeskCfg `mapstructure:"zendesk" json:"zendesk"`
	CW      cwCfg      `mapstructure:"connectwise_psa" json:"connectwise_psa"`
}

type zendeskCfg struct {
	ApiCreds zendesk.Creds `mapstructure:"api_creds" json:"api_creds"`
	FieldIds fieldIds      `mapstructure:"field_ids" json:"field_ids"`
}

type fieldIds struct {
	PSACompanyId string `mapstructure:"psa_company_id" json:"psa_company_id"`
	PSAContactId string `mapstructure:"psa_contact_id" json:"psa_contact_id"`
	PSATicketId  int    `mapstructure:"psa_ticket_id" json:"psa_ticket_id"`
}

type cwCfg struct {
	ApiCreds           cw.Creds `mapstructure:"api_creds" json:"api_creds"`
	ClosedStatusId     int      `mapstructure:"closed_status_id" json:"closed_status_id"`
	OpenStatusId       int      `mapstructure:"open_status_id" json:"open_status_id"`
	DestinationBoardId int      `mapstructure:"destination_board_id" json:"destination_board_id"`
}

var configCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"cfg"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := conf.interactiveForm().Run(); err != nil {
			return err
		}

		viper.Set("zendesk", conf.Zendesk)
		viper.Set("connectwise_psa", conf.CW)

		if err := viper.WriteConfig(); err != nil {
			return err
		}

		if testConn {
			client = migration.NewClient(conf.Zendesk.ApiCreds, conf.CW.ApiCreds)
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
			//fmt.Println("cfg file created - location:", path)
			//fmt.Println("Please fill in the necessary fields and run the program again.")
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

func (cfg *cfg) interactiveForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Zendesk Token").
				Placeholder(cfg.Zendesk.ApiCreds.Token).
				Value(&cfg.Zendesk.ApiCreds.Token),
			huh.NewInput().
				Title("Zendesk Username").
				Placeholder(cfg.Zendesk.ApiCreds.Username).
				Value(&cfg.Zendesk.ApiCreds.Username),
			huh.NewInput().
				Title("Zendesk Subdomain").
				Placeholder(cfg.Zendesk.ApiCreds.Subdomain).
				Value(&cfg.Zendesk.ApiCreds.Subdomain),
			huh.NewInput().
				Title("ConnectWise Company ID").
				Placeholder(cfg.Zendesk.FieldIds.PSACompanyId).
				Value(&cfg.CW.ApiCreds.CompanyId),
			huh.NewInput().
				Title("ConnectWise Public Key").
				Placeholder(cfg.Zendesk.FieldIds.PSAContactId).
				Value(&cfg.CW.ApiCreds.PublicKey),
			huh.NewInput().
				Title("ConnectWise Private Key").
				Placeholder(cfg.CW.ApiCreds.CompanyId).
				Value(&cfg.CW.ApiCreds.PrivateKey),
			huh.NewInput().
				Title("ConnectWise Client ID").
				Placeholder(cfg.CW.ApiCreds.ClientId).
				Value(&cfg.CW.ApiCreds.ClientId),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Test API Connection?").
				Value(&testConn),
		),
	).WithTheme(huh.ThemeBase())
}

func setCfgDefaults() {
	slog.Debug("setting config defaults")
	//viper.SetDefault("agentMappings", []migration.Agent{{}}) // prefill with empty agent
	viper.SetDefault("zendesk", zendeskCfg{})
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
