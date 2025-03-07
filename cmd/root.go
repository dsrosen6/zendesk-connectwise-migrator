package cmd

import (
	"context"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/migration"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log/slog"
	"os"
)

var (
	verbose bool
	ctx     context.Context
	client  *migration.Client
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
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	
	rootCmd.AddCommand(testCmd)
	cobra.OnInitialize(initConfig)
}

func setLogger(v bool) *slog.Logger {
	level := slog.LevelInfo
	if v {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	return logger
}

func preRun(cmd *cobra.Command, args []string) error {
	ctx = context.Background()
	slog.SetDefault(setLogger(verbose))

	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("unmarshaling config: %w", err)
	}

	if err := validateConfig(config); err != nil {
		return fmt.Errorf("config validation: %w", err)
	}

	client = migration.NewClient(config.Zendesk.ApiCreds, config.CW.ApiCreds)
	return nil
}
