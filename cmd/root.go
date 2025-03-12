package cmd

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration/tui"
	"github.com/spf13/cobra"
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
		return nil
	},
}

var runCmd = &cobra.Command{
	Use:          "run",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := migration.RunStartup(ctx)
		if err != nil {
			return fmt.Errorf("running startup: %w", err)
		}

		p := tea.NewProgram(tui.NewModel(ctx, client))
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("an error occured launching the terminal interface: %w", err)
		}

		return nil
	},
}

var cfgCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"cfg"},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := migration.InitConfig()
		if err != nil {
			return fmt.Errorf("initializing config: %w", err)
		}

		if err := cfg.PromptAllFields(); err != nil {
			return fmt.Errorf("prompting fields: %w", err)
		}

		fmt.Println("Config saved")
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
	rootCmd.AddCommand(cfgCmd)
	rootCmd.AddCommand(runCmd)
}
