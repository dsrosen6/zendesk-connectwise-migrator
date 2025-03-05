package cmd

import (
	"errors"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/zendesk"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	configFileSubPath = "/migrator_config.json"
)

var cfgFile string

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
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(viper.GetViper().GetString("test"))
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
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.zendesk-connectwise-migrator.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {

		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			fmt.Println("Config file not found - creating default. Please edit the file at ~/migrator_config.json")
			setCfgDefaults()
			fmt.Println("Writing default config file to: ", home)
			if err := viper.WriteConfigAs(home + configFileSubPath); err != nil {
				fmt.Println("Error writing config file: ", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("Error reading config file: ", err)
			os.Exit(1)
		}
	}
}

func setCfgDefaults() {
	viper.SetDefault("zendesk", map[string]any{
		"api_creds": zendesk.Creds{},
	})

	viper.SetDefault("connectwise_psa", map[string]any{
		"api_creds":            psa.Creds{},
		"destination_board_id": "",
		"open_status_id":       "",
		"closed_status_id":     "",
	})

	viper.SetDefault("agent_mappings", []migration.Agent{{}, {}}) // prefill with empty agents
}
