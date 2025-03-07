package cmd

import (
	"context"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/migration"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"path/filepath"
)

var (
	ctx    context.Context
	client *migration.Client
	debug  bool
)

var rootCmd = &cobra.Command{
	Use:               "zendesk-connectwise-migrator",
	SilenceUsage:      true,
	PersistentPreRunE: preRun,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.zendesk-connectwise-migrator.yaml)")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "enable debug logging")
	rootCmd.AddCommand(testCmd)
	cobra.OnInitialize(initConfig)
}

func preRun(cmd *cobra.Command, args []string) error {
	ctx = context.Background()
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	file, err := openLogFile(filepath.Join(home, "migrator.log"))
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	debug, err = cmd.Flags().GetBool("debug")
	if err != nil {
		return fmt.Errorf("getting debug flag: %w", err)
	}

	if err := setLogger(file); err != nil {
		return fmt.Errorf("setting logger: %w", err)
	}

	slog.Info("starting migrator CLI pre-run processes")
	if err := viper.Unmarshal(&config); err != nil {
		slog.Error("unmarshaling config", "error", err)
		return fmt.Errorf("unmarshaling config: %w", err)
	}

	if err := validateConfig(config); err != nil {
		slog.Error("validating config", "error", err)
		return fmt.Errorf("config validation: %w", err)
	}
	
	client = migration.NewClient(config.Zendesk.ApiCreds, config.CW.ApiCreds)
	return nil
}
