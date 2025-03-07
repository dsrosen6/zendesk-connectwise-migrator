package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"log/slog"
	"strconv"
)

var testCmd = &cobra.Command{
	Use: "test",
}

var testConnectionCmd = &cobra.Command{
	Use: "connection",
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

var inputCmd = &cobra.Command{
	Use:     "input-ticket",
	Aliases: []string{"i"},
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
		fmt.Println("Organization:", i.Organization.Organization.Name)
		fmt.Println("Requester:", i.Requester.User.Name)
		fmt.Println("Assignee:", i.Assignee.User.Name)
		fmt.Println("Total Comments:", len(i.Comments))
		fmt.Println("Closed:", i.Closed)
		if i.Closed {
			fmt.Println("Closed At:", i.ClosedAt)
		}
		return nil
	},
}

var getCmd = &cobra.Command{
	Use: "get",
}

// gets orgs by tags
var getOrgsCmd = &cobra.Command{
	Use:  "orgs",
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tags := args
		orgs, err := client.ZendeskClient.GetOrganizationsWithQuery(ctx, tags)
		if err != nil {
			return fmt.Errorf("an error occured getting organizations: %w", err)
		}

		if len(orgs) == 0 {
			fmt.Println("No organizations found")
			return nil
		}

		slog.Debug("getOrgsCmd", "total_orgs", len(orgs))
		fmt.Println("Orgs found:")
		for _, o := range orgs {
			fmt.Printf("   %s\n", o.Organization.Name)
		}

		return nil
	},
}

var matchCmd = &cobra.Command{
	Use: "match",
}

var matchCompanyCmd = &cobra.Command{
	Use: "org",
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

		m, err := client.MatchZdOrgToCwCompany(ctx, i.Organization)
		if err != nil {
			return fmt.Errorf("an error occured matching org to company: %w", err)
		}

		fmt.Println("Matched Company:", m.Name)
		fmt.Println("Matched Company ID:", m.Id)
		return nil
	},
}

func init() {
	testCmd.AddCommand(testConnectionCmd)
	testCmd.AddCommand(inputCmd)
	testCmd.AddCommand(getCmd)
	testCmd.AddCommand(matchCmd)

	getCmd.AddCommand(getOrgsCmd)

	matchCmd.AddCommand(matchCompanyCmd)
}
