package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

// matchCmd represents the match command
var matchCmd = &cobra.Command{
	Use: "match",
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
		
		m, err := client.MatchOrgToCompany(ctx, i.Organization)
		if err != nil {
			return fmt.Errorf("an error occured matching org to company: %w", err)
		}

		fmt.Println("Matched Company:", m.Name)
		fmt.Println("Matched Company ID:", m.Id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(matchCmd)
}
