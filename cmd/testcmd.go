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

		slog.Debug("testing Zendesk connection")
		if err := client.ZendeskClient.TestConnection(ctx); err != nil {
			slog.Error("testConnectionCmd", "action", "client.ZendeskClient.TestConnection", "error", err)
			return fmt.Errorf("zendesk connection test failed: %w", err)
		}
		slog.Info("successfully connected to Zendesk")
		fmt.Println("Zendesk connection test successful")

		slog.Debug("testing ConnectWise connection")
		if err := client.CwClient.TestConnection(ctx); err != nil {
			slog.Error("testConnectionCmd", "action", "client.CwClient.TestConnection", "error", err)
			return fmt.Errorf("connectwise connection test failed: %w", err)
		}

		slog.Info("successfully connected to ConnectWise")
		fmt.Println("ConnectWise connection test successful")
		return nil
	},
}

var inputCmd = &cobra.Command{
	Use:     "input-ticket",
	Aliases: []string{"i"},
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Info("running inputCmd", "args", args)
		if len(args) != 1 {
			slog.Error("inputCmd", "error", "expected exactly one argument (ticket ID)")
			return fmt.Errorf("expected exactly one argument (ticket ID)")
		}
		ticketId := args[0]
		slog.Debug("converting ticket ID to int", "ticket_id", ticketId)
		ti, err := strconv.Atoi(args[0])
		if err != nil {
			slog.Error("inputCmd", "error", "expected exactly one argument (ticket ID)")
			return fmt.Errorf("invalid ticket ID - must be an integer")
		}

		slog.Debug("constructing input ticket", "ticket_id", ti)
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
		slog.Info("running getOrgsCmd", "args", args)
		tags := args
		orgs, err := client.ZendeskClient.GetOrganizationsWithQuery(ctx, tags)
		if err != nil {
			slog.Error("getOrgsCmd", "action", "client.ZendeskClient.GetOrganizationsWithQuery", "error", err)
			return fmt.Errorf("an error occured getting organizations: %w", err)
		}

		if len(orgs) == 0 {
			slog.Debug("no orgs found")
			fmt.Println("No organizations found")
			return nil
		}

		fmt.Println("Orgs found:")
		for _, o := range orgs {
			fmt.Println("Name:", o.Name)
		}

		slog.Info("getOrgsCmd", "action", "client.ZendeskClient.GetOrganizationsWithQuery", "total_orgs", len(orgs))
		return nil
	},
}

var matchCmd = &cobra.Command{
	Use: "match",
}

func init() {
	testCmd.AddCommand(testConnectionCmd)
	testCmd.AddCommand(inputCmd)
	testCmd.AddCommand(getCmd)
	testCmd.AddCommand(matchCmd)

	getCmd.AddCommand(getOrgsCmd)
}
