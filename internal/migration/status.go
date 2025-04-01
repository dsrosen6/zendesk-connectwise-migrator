package migration

import tea "github.com/charmbracelet/bubbletea"

type migrationStatus string

const (
	awaitingStart      migrationStatus = "Awaiting Start"
	gettingTags        migrationStatus = "Getting Tags from Config"
	gettingZendeskOrgs migrationStatus = "Getting Zendesk Organizations"
	comparingOrgs      migrationStatus = "Checking for Organization Matches"
	initOrgForm        migrationStatus = "Initializing Form"
	pickingOrgs        migrationStatus = "Selecting Organizations"
	gettingUsers       migrationStatus = "Getting Users"
	migratingUsers     migrationStatus = "Migrating Users"
	gettingPsaTickets  migrationStatus = "Getting PSA Tickets"
	migratingTickets   migrationStatus = "Migrating Tickets"
	done               migrationStatus = "Done"
	errored            migrationStatus = "Error"
)

type switchStatusMsg migrationStatus

func switchStatus(s migrationStatus) tea.Cmd {
	return func() tea.Msg { return switchStatusMsg(s) }
}

type ticketStatus string

const (
	ticketStatusGetting   ticketStatus = "ticketStatusGetting"
	ticketStatusMigrating ticketStatus = "ticketStatusMigrating"
)
