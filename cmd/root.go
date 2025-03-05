package cmd

import (
	"errors"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/cw"
	"github.com/dsrosen/zendesk-connectwise-migrator/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/zendesk"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	configFileSubPath = "/migrator_config.json"
)

var cfgFile string
var verbose bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "zendesk-connectwise-migrator",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		slog.SetDefault(setLogger(verbose))
	},
	Run: func(cmd *cobra.Command, args []string) {
		missing := verifyConfigSet()
		if len(missing) > 0 {
			fmt.Println("Missing config fields:", missing)
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.zendesk-connectwise-migrator.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
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

	viper.SetDefault("connectwise_psa", map[string]any{
		"api_creds":            cw.Creds{},
		"destination_board_id": 0,
		"open_status_id":       0,
		"closed_status_id":     0,
	})

	viper.SetDefault("agent_mappings", []migration.Agent{{}, {}}) // prefill with empty agents
	viper.SetDefault("debug", false)
}

func setLogger(v bool) *slog.Logger {
	level := slog.LevelInfo
	if v {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	return logger
}

func verifyConfigSet() []string {
	var missing []string

	keysWithStrVal := []string{
		"zendesk.api_creds.token",
		"zendesk.api_creds.username",
		"zendesk.api_creds.subdomain",
		"connectwise_psa.api_creds.company_id",
		"connectwise_psa.api_creds.public_key",
		"connectwise_psa.api_creds.private_key",
		"connectwise_psa.api_creds.client_id",
	}

	keysWithIntVal := []string{
		"connectwise_psa.destination_board_id",
		"connectwise_psa.open_status_id",
		"connectwise_psa.closed_status_id",
	}

	for _, key := range keysWithStrVal {
		if viper.GetString(key) == "" {
			missing = append(missing, key)
		}
	}

	for _, key := range keysWithIntVal {
		if viper.GetInt(key) == 0 {
			missing = append(missing, key)
		}
	}

	return missing
}
