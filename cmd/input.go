package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"strconv"
)

var inputCmd = &cobra.Command{
	Use:     "get-input-ticket",
	Aliases: []string{"ip"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("expected exactly one argument (ticket ID)")
		}

		ti, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid ticket ID - must be an integer")
		}

		i, err := client.ConstructInputTicket(ctx, int64(ti))
		if err != nil {
			return fmt.Errorf("an error occured constructing input ticket: %w", err)
		}

		fmt.Println("Subject:", i.Subject)
		fmt.Println("Requester:", i.Requester.User.Name)
		fmt.Println("Assignee:", i.Assignee.User.Name)
		fmt.Println("Total Comments:", len(i.Comments))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(inputCmd)
}
