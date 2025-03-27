package cmd

import (
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/spf13/cobra"
	"os"
)

var debug bool

var rootCmd = &cobra.Command{
	Use:          "zendesk-connectwise-migrator",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		debug, err = cmd.Flags().GetBool("debug")
		if err != nil {
			return fmt.Errorf("getting debug flag: %w", err)
		}

		return migration.Run(debug)
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
