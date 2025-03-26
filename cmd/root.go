package cmd

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/tui"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
	"path/filepath"
)

var (
	ctx   context.Context
	debug bool
)

var rootCmd = &cobra.Command{
	Use:          "zendesk-connectwise-migrator",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx = context.Background()
		dir, err := makeMigrationDir()
		if err != nil {
			return fmt.Errorf("creating migration directory: %w", err)
		}

		logFile, err := openLogFile(filepath.Join(dir, "migration.log"))
		if err != nil {
			return fmt.Errorf("opening log file: %w", err)
		}

		debug, err = cmd.Flags().GetBool("debug")
		if err != nil {
			return fmt.Errorf("getting debug flag: %w", err)
		}

		if err := setLogger(logFile); err != nil {
			return fmt.Errorf("setting logger: %w", err)
		}

		client, err := migration.RunStartup(ctx, dir)
		if err != nil {
			slog.Error("running startup", "error", err)
			return err
		}

		model, err := tui.NewModel(ctx, client)
		if err != nil {
			return fmt.Errorf("initializing terminal interface: %w", err)
		}

		p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
		if _, err := p.Run(); err != nil {
			slog.Error("running terminal interface", "error", err)
			return fmt.Errorf("launching terminal interface: %w", err)
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
}

func makeMigrationDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting user home directory: %w", err)
	}

	migrationDir := filepath.Join(home, "ticket-migration")
	if err := os.MkdirAll(migrationDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("creating migration directory: %w", err)
	}

	return migrationDir, nil
}
