package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

// testConnectionCmd represents the testConnection command
var testConnectionCmd = &cobra.Command{
	Use: "testconnection",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.ZendeskClient.TestConnection(ctx); err != nil {
			return fmt.Errorf("zendesk connection test failed: %w", err)
		}

		fmt.Println("Zendesk connection test successful")

		if err := client.CwClient.TestConnection(ctx); err != nil {
			return fmt.Errorf("connectwise connection test failed: %w", err)
		}

		fmt.Println("ConnectWise connection test successful")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(testConnectionCmd)
}
