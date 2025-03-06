package cmd

import (
	"errors"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/cw"
	"github.com/dsrosen/zendesk-connectwise-migrator/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/zendesk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	AgentMappings []migration.Agent `mapstructure:"agentMappings"`
	Zendesk       ZendeskConfig     `mapstructure:"zendesk"`
	CW            CwConfig          `mapstructure:"connectwisePsa"`
	Debug         bool              `mapstructure:"debug"`
}

type ZendeskConfig struct {
	ApiCreds zendesk.Creds `mapstructure:"apiCreds"`
}

type CwConfig struct {
	ApiCreds           cw.Creds `mapstructure:"apiCreds"`
	ClosedStatusId     int      `mapstructure:"closedStatusId"`
	OpenStatusId       int      `mapstructure:"openStatusId"`
	DestinationBoardId int      `mapstructure:"destinationBoardId"`
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
	viper.SetDefault("zendesk", map[string]any{
		"api_creds": zendesk.Creds{},
	})

	viper.SetDefault("connectwisePsa", map[string]any{
		"apiCreds":           cw.Creds{},
		"destinationBoardId": 0,
		"openStatusId":       0,
		"closedStatusId":     0,
	})

	viper.SetDefault("agentMappings", []migration.Agent{{}, {}}) // prefill with empty agents
	viper.SetDefault("debug", false)
}

func validateConfig(cfg Config) error {
	if err := validateRequiredValues(cfg); err != nil {
		return err
	}

	if err := validateAgentMappings(cfg.AgentMappings); err != nil {
		return err
	}

	return nil
}

func validateRequiredValues(cfg Config) error {
	var missing []string

	keysWithStrVal := []string{
		cfg.Zendesk.ApiCreds.Token,
		cfg.Zendesk.ApiCreds.Username,
		cfg.Zendesk.ApiCreds.Subdomain,
		cfg.CW.ApiCreds.CompanyId,
		cfg.CW.ApiCreds.PublicKey,
		cfg.CW.ApiCreds.PrivateKey,
		cfg.CW.ApiCreds.ClientId,
	}

	keysWithIntVal := []int{
		cfg.CW.ClosedStatusId,
		cfg.CW.OpenStatusId,
		cfg.CW.DestinationBoardId,
	}

	// if value is empty, add key to missing
	for _, key := range keysWithStrVal {
		if key == "" {
			missing = append(missing, key)
		}
	}

	// if value is 0, add key to missing
	for _, key := range keysWithIntVal {
		if key == 0 {
			missing = append(missing, fmt.Sprintf("%d", key))
		}
	}

	if len(missing) > 0 {
		return errors.New("missing 1 or more required config values")
	}

	return nil
}

func validateAgentMappings(agents []migration.Agent) error {
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
			fmt.Printf("\nAn agent mapping is missing: %s\n   current values:\n       name: %s\n       zendeskUserId: %d\n       connectwiseMemberId: %d\n",
				strings.Join(missingAgentVals, ","), agent.Name, agent.ZendeskId, agent.CwId)

			missingAgent = true
		}
	}

	if missingAgent {
		return errors.New("missing agent mapping")
	}

	return nil
}
