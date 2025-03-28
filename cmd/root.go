package cmd

import (
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:          "zendesk-connectwise-migrator",
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
	rootCmd.PersistentFlags().Bool("cutNoAction", false, "don't show output of items with no action")
	rootCmd.PersistentFlags().Bool("cutCreated", false, "don't show output of items created")
	rootCmd.PersistentFlags().Bool("cutWarn", false, "don't show output of items with warnings")
	rootCmd.PersistentFlags().Bool("cutError", false, "don't show output of items with errors")
	rootCmd.PersistentFlags().Bool("onlyErrors", false, "only show output of items with errors")
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

	onlyErrors, err := cmd.Flags().GetBool("onlyErrors")
	if err != nil {
		return migration.CliOptions{}, fmt.Errorf("getting only errors flag: %w", err)
	}

	if onlyErrors {
		return migration.CliOptions{
			Debug:              debug,
			TicketLimit:        ticketLimit,
			MigrateOpenTickets: migrateOpen,
			OutputLevels: migration.OutputLevels{
				NoAction: false,
				Created:  false,
				Warn:     false,
				Error:    true,
			},
		}, nil
		
	} else {
		cutNoAction, err := cmd.Flags().GetBool("cutNoAction")
		if err != nil {
			return migration.CliOptions{}, fmt.Errorf("getting cut no action flag: %w", err)
		}

		cutCreated, err := cmd.Flags().GetBool("cutCreated")
		if err != nil {
			return migration.CliOptions{}, fmt.Errorf("getting cut created flag: %w", err)
		}

		cutWarn, err := cmd.Flags().GetBool("cutWarn")
		if err != nil {
			return migration.CliOptions{}, fmt.Errorf("getting cut warn flag: %w", err)
		}

		cutError, err := cmd.Flags().GetBool("cutError")
		if err != nil {
			return migration.CliOptions{}, fmt.Errorf("getting cut error flag: %w", err)
		}

		return migration.CliOptions{
			Debug:              debug,
			TicketLimit:        ticketLimit,
			MigrateOpenTickets: migrateOpen,
			OutputLevels: migration.OutputLevels{
				NoAction: !cutNoAction,
				Created:  !cutCreated,
				Warn:     !cutWarn,
				Error:    !cutError,
			},
		}, nil
	}
}
