package cmd

import (
	"context"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
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
	Use:          "zendesk-connectwise-migrator",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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

		if err := viper.Unmarshal(&conf); err != nil {
			slog.Error("unmarshaling config", "error", err)
			return fmt.Errorf("unmarshaling config: %w", err)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && args[0] == "config" {
			return conf.runCredsForm()
		}

		if err := conf.validateConfig(); err != nil {
			if err := conf.runCredsForm(); err != nil {
				return fmt.Errorf("validating config: %w", err)
			}
		}

		client = migration.NewClient(conf.Zendesk.Creds, conf.CW.Creds)

		if err := client.ConnectionTest(ctx); err != nil {
			return fmt.Errorf("connection test: %w", err)
		}

		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "enable debug logging")
	rootCmd.AddCommand(testCmd)
	cobra.OnInitialize(initConfig)
}
