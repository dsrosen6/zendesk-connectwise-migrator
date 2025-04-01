package cmd

import (
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:          "migrator",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := parseFlags(cmd)
		if err != nil {
			return fmt.Errorf("parsing flags: %w", err)
		}

		return migration.Run(opts)
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
	rootCmd.PersistentFlags().IntP("ticketLimit", "t", 0, "limit the number of tickets to migrate")
	rootCmd.PersistentFlags().BoolP("migrateOpen", "o", false, "migrate open and closed tickets (default is closed only)")
	rootCmd.PersistentFlags().Bool("showNoAction", false, "show output of items with no action (default is false)")
	rootCmd.PersistentFlags().Bool("showCreated", false, "show output of items created (default is false)")
	rootCmd.PersistentFlags().Bool("showWarn", true, "show output of items with warnings (default is true)")
	rootCmd.PersistentFlags().Bool("showError", true, "show output of items with errors (default is true)")
	rootCmd.PersistentFlags().Bool("stopAfterOrgs", false, "stop migration after getting orgs")
	rootCmd.PersistentFlags().Bool("stopAfterUsers", false, "stop migration after getting users")
	rootCmd.PersistentFlags().Bool("stopAtError", false, "stop migration after first error")
}

func parseFlags(cmd *cobra.Command) (migration.CliOptions, error) {
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return migration.CliOptions{}, fmt.Errorf("getting debug flag: %w", err)
	}

	ticketLimit, err := cmd.Flags().GetInt("ticketLimit")
	if err != nil {
		return migration.CliOptions{}, fmt.Errorf("getting ticket limit flag: %w", err)
	}

	migrateOpen, err := cmd.Flags().GetBool("migrateOpen")
	if err != nil {
		return migration.CliOptions{}, fmt.Errorf("getting migrate open flag: %w", err)
	}

	stopAfterOrgs, err := cmd.Flags().GetBool("stopAfterOrgs")
	if err != nil {
		return migration.CliOptions{}, fmt.Errorf("getting stop after orgs flag: %w", err)
	}

	stopAfterUsers, err := cmd.Flags().GetBool("stopAfterUsers")
	if err != nil {
		return migration.CliOptions{}, fmt.Errorf("getting stop after users flag: %w", err)
	}

	showNoAction, err := cmd.Flags().GetBool("showNoAction")
	if err != nil {
		return migration.CliOptions{}, fmt.Errorf("getting show no action flag: %w", err)
	}

	showCreated, err := cmd.Flags().GetBool("showCreated")
	if err != nil {
		return migration.CliOptions{}, fmt.Errorf("getting show created flag: %w", err)
	}

	showWarn, err := cmd.Flags().GetBool("showWarn")
	if err != nil {
		return migration.CliOptions{}, fmt.Errorf("getting show warn flag: %w", err)
	}

	showError, err := cmd.Flags().GetBool("showError")
	if err != nil {
		return migration.CliOptions{}, fmt.Errorf("getting show error flag: %w", err)
	}

	stopAtError, err := cmd.Flags().GetBool("stopAtError")
	if err != nil {
		return migration.CliOptions{}, fmt.Errorf("getting stop at error flag: %w", err)
	}

	return migration.CliOptions{
		Debug:              debug,
		TicketLimit:        ticketLimit,
		MigrateOpenTickets: migrateOpen,
		OutputLevels: migration.OutputLevels{
			NoAction: showNoAction,
			Created:  showCreated,
			Warn:     showWarn,
			Error:    showError,
		},
		StopAfterOrgs:  stopAfterOrgs,
		StopAfterUsers: stopAfterUsers,
		StopAtError:    stopAtError,
	}, nil
}
