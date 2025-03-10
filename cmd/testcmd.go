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
		slog.Info("running testConnectionCmd")
		return client.ConnectionTest(ctx)
	},
}

var inputCmd = &cobra.Command{
	Use:     "input-ticket",
	Aliases: []string{"i"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			slog.Error("inputCmd", "error", "expected exactly one argument (ticket ID)")
			return fmt.Errorf("expected exactly one argument (ticket ID)")
		}
		ticketId := args[0]
		ti, err := strconv.Atoi(ticketId)
		if err != nil {
			slog.Error("inputCmd", "error", "expected exactly one argument (ticket ID)")
			return fmt.Errorf("invalid ticket ID - must be an integer")
		}

		i, err := client.ConstructInputTicket(ctx, int64(ti))
		if err != nil {
			slog.Error("inputCmd", "action", "client.ConstructInputTicket", "error", err)
			return fmt.Errorf("an error occured constructing input ticket: %w", err)
		}

		fmt.Println("Subject:", i.Subject)
		fmt.Println("Organization:", i.Organization.Name)
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
			slog.Error("getOrgsCmd", "action", "client.ZendeskClient.GetOrganizationsWithQuery", "error", err)
			return fmt.Errorf("an error occured getting organizations: %w", err)
		}

		if len(orgs) == 0 {
			fmt.Println("No organizations found")
			return nil
		}

		fmt.Println("Orgs found:")
		for _, o := range orgs {
			fmt.Println("Name:", o.Name)
		}

		return nil
	},
}

var matchCmd = &cobra.Command{
	Use:  "match",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email := args[0]
		u, err := client.CwClient.GetContactByEmail(ctx, email)
		if err != nil {
			slog.Error("matchCmd", "action", "client.CwClient.GetContactByEmail", "error", err)
			return fmt.Errorf("an error occured getting contact by email: %w", err)
		}

		if u == nil {
			fmt.Println("No contact found")
			return nil
		}

		fmt.Println("Contact found:", u.Id)
		return nil
	},
}

var migrateUserCmd = &cobra.Command{
	Use:  "migrate-user",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		zendeskUserId := args[0]
		psaCompanyId := args[1]

		zi, err := strconv.Atoi(zendeskUserId)
		if err != nil {
			slog.Error("migrateUserCmd", "error", "invalid zendeskUserId")
			return fmt.Errorf("invalid zendeskUserId - must be an integer")
		}

		zu, err := client.ZendeskClient.GetUser(ctx, int64(zi))
		if err != nil {
			slog.Error("migrateUserCmd", "error", "error getting user")
			return fmt.Errorf("error getting user: %w", err)
		}

		pi, err := strconv.Atoi(psaCompanyId)
		if err != nil {
			slog.Error("migrateUserCmd", "error", "invalid psaCompanyId")
			return fmt.Errorf("invalid psaCompanyId - must be an integer")
		}

		_, err = client.MigrateUser(ctx, zu, pi)
		if err != nil {
			slog.Error("migrateUserCmd", "error", "error migrating user")
			return fmt.Errorf("error migrating user: %w", err)
		}

		fmt.Println("User migrated")
		return nil
	},
}

func init() {
	testCmd.AddCommand(testConnectionCmd)
	testCmd.AddCommand(inputCmd)
	testCmd.AddCommand(getCmd)
	testCmd.AddCommand(matchCmd)
	testCmd.AddCommand(migrateUserCmd)

	getCmd.AddCommand(getOrgsCmd)
}
