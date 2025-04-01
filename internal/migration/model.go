package migration

import (
	"context"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"log/slog"
	"regexp"
	"sync"
	"time"
)

type Model struct {
	// API
	mu     sync.Mutex
	ctx    context.Context
	client *Client

	// Migration State
	timeZone               *time.Location
	form                   *huh.Form
	formComplete           bool
	allOrgsSelected        bool
	status                 migrationStatus
	currentTicketMigration *activeTicketMigration
	data                   *Data
	statistics

	// UI
	viewport viewport.Model
	spinner  spinner.Model
	dimensions
	viewState
	scrollManagement
}

type outputLevel string

const (
	noActionOutput outputLevel = "noActionOutput"
	createdOutput  outputLevel = "createdActionOutput"
	warnOutput     outputLevel = "warnActionOutput"
	errOutput      outputLevel = "errorActionOutput"
)

type statistics struct {
	orgsChecked         int
	orgsNotInPsa        int
	orgsMigrated        int
	orgsCheckedForUsers int
	usersProcessed      int
	newUsersCreated     int
	ticketsToProcess    int
	ticketsProcessed    int
	newTicketsCreated   int
	ticketOrgsProcessed int

	userMigrationErrors   int
	ticketMigrationErrors int
}

type viewState struct {
	ready    bool
	quitting bool
}

func newModel(ctx context.Context, client *Client) (*Model, error) {
	spnr := spinner.New()
	spnr.Spinner = spinner.Ellipsis
	spnr.Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})

	data := client.newData()

	if client.Cfg.TicketLimit > 0 {
		slog.Info("ticket test limit in config", "limit", client.Cfg.TicketLimit)
	}

	loc, err := client.getTimeZone()
	if err != nil {
		slog.Error("getting time zone", "error", err)
		return nil, fmt.Errorf("getting time zone: %w", err)
	}

	slog.Info("time zone set", "timeZone", loc.String())

	return &Model{
		ctx:                    ctx,
		client:                 client,
		data:                   data,
		status:                 awaitingStart,
		timeZone:               loc,
		spinner:                spnr,
		currentTicketMigration: newActiveTicketMigration("none", ticketStatusGetting),
	}, nil
}

func (m *Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		slog.Debug("got WindowSizeMsg")
		return m, m.calculateDimensions(msg.Width, msg.Height)

	case tea.KeyMsg:

		switch msg.String() {
		case "ctrl+q":
			m.quitting = true
			cmds = append(cmds, tea.Quit)
		case "c":
			cmds = append(cmds, m.copyToClipboard(m.data.Output.String()))
		case " ":
			if m.status == awaitingStart {
				return m, switchStatus(gettingTags)
			}
		}

	case tea.MouseMsg:
		// override automatic scrolling if user scrolls up or down so they can read output
		// if they scroll back to the bottom, resume automatic scrolling to the bottom (see
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			m.scrollOverride = true
		}

	case switchStatusMsg:
		m.status = migrationStatus(msg)
		switch migrationStatus(msg) {
		case gettingTags:
			slog.Debug("getting tags from config")
			return m, m.getTagDetails()
		case gettingZendeskOrgs:
			slog.Debug("getting zendesk orgs")
			m.statistics = statistics{}
			return m, m.getOrgs()
		case comparingOrgs:
			slog.Debug("comparing orgs")
			var checkOrgCmds []tea.Cmd
			for _, org := range m.data.AllOrgs {
				checkOrgCmds = append(checkOrgCmds, m.checkOrg(org))
			}

			return m, tea.Batch(checkOrgCmds...)
		case initOrgForm:
			slog.Debug("initializing org form")
			m.form = m.orgSelectionForm()
			cmds = append(cmds, m.form.Init(), switchStatus(pickingOrgs))
			return m, tea.Sequence(cmds...)
		case gettingUsers:
			slog.Debug("getting users for all selected orgs")
			m.data.UsersToMigrate = make(map[string]*userMigrationDetails)
			var batches []tea.Cmd
			var currentBatch []tea.Cmd
			const batchSize = 20

			for _, org := range m.data.SelectedOrgs {
				currentBatch = append(currentBatch, m.getUsersToMigrate(org))
				if len(currentBatch) == batchSize {
					batches = append(batches, tea.Batch(currentBatch...))
					currentBatch = nil
				}
			}

			if len(currentBatch) > 0 {
				batches = append(batches, tea.Batch(currentBatch...))
			}
			cmds = append(cmds, tea.Batch(batches...))
			return m, tea.Sequence(cmds...)
		case migratingUsers:
			slog.Debug("migrating users")
			cmds = append(cmds, m.migrateUsers(m.data.UsersToMigrate))
			return m, tea.Sequence(cmds...)

		case gettingPsaTickets:
			return m, m.getAlreadyMigrated()

		case migratingTickets:
			for _, org := range m.data.SelectedOrgs {
				cmds = append(cmds, m.runTicketMigration(org))
			}

			return m, tea.Sequence(cmds...)
		}

	case fatalErrMsg:
		slog.Error("fatal error", "error", msg.Err)
		cmds = append(cmds, switchStatus(done))
	}

	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	switch m.status {
	case comparingOrgs:
		if len(m.data.AllOrgs) == m.orgsChecked {
			if m.client.Cfg.StopAfterOrgs {
				slog.Info("stopping after org check as per configuration")
				cmds = append(cmds, switchStatus(done))
			} else {
				cmds = append(cmds, switchStatus(initOrgForm))
			}
		}

	case pickingOrgs:
		form, cmd := m.form.Update(msg)
		cmds = append(cmds, cmd)

		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		if m.form.State == huh.StateCompleted && !m.formComplete {
			switch m.status {
			case pickingOrgs:
				if m.allOrgsSelected {
					for _, org := range m.data.AllOrgs {
						if !org.Migrated {
							continue
						}
						m.data.SelectedOrgs = append(m.data.SelectedOrgs, org)
					}
				}

				slog.Debug("form completed, selected orgs", "selectedOrgsCount", len(m.data.SelectedOrgs))
				m.formComplete = true
				cmds = append(cmds, switchStatus(gettingUsers))
			}
		}

	case gettingUsers:
		if len(m.data.SelectedOrgs) == m.orgsCheckedForUsers {
			cmds = append(cmds, switchStatus(migratingUsers))
		}

	case migratingTickets:
		if len(m.data.SelectedOrgs) == m.ticketOrgsProcessed {
			cmds = append(cmds, switchStatus(done))
		}
	}

	if m.ready {
		m.viewport.SetContent(m.data.Output.String())
		m.setAutoScrollBehavior()
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if m.quitting {
		return ""
	}

	if !m.ready {
		return m.runSpinner("Initializing")
	}

	var s string
	switch m.status {
	case awaitingStart:
		s += welcomeText()
	case comparingOrgs:
		s += m.runSpinner(fmt.Sprintf("Checking organizations (%d/%d)", m.orgsChecked, len(m.data.AllOrgs)))
	case gettingUsers:
		s += m.runSpinner(fmt.Sprintf("Getting users for all selected orgs - got %d users", len(m.data.UsersToMigrate)))
	case migratingUsers:
		s += m.runSpinner(fmt.Sprintf("Migrating users (%d/%d)", m.usersProcessed, len(m.data.UsersToMigrate)))
	case pickingOrgs:
		s += m.form.View()
	case gettingPsaTickets:
		s += m.runSpinner("Getting existing tickets from the PSA")
	case migratingTickets:
		switch m.currentTicketMigration.status {
		case ticketStatusGetting:
			s += m.runSpinner(fmt.Sprintf("Getting Zendesk tickets for org %s", m.currentTicketMigration.orgName))
		case ticketStatusMigrating:
			s += m.runSpinner(fmt.Sprintf("Migrating tickets for org %s - %d/%d done", m.currentTicketMigration.orgName, m.currentTicketMigration.ticketsProcessed, m.currentTicketMigration.ticketsToProcess))
		default:
			s += m.runSpinner("Starting ticket migration")
		}
	case done:
		s += "Migration complete - press CTRL+Q to exit.\n\nTo run the migration again, exit and run the utility again."
	default:
		s += m.runSpinner(string(m.status))
	}

	if m.status != awaitingStart && m.status != pickingOrgs {
		s += fmt.Sprintf("\n\nUsers Processed: %d\n"+
			"New Users Created: %d\n"+
			"Tickets Processed: %d\n"+
			"New Tickets Created: %d\n"+
			"Orgs Complete: %d/%d\n"+
			"Orgs Not in PSA: %d\n"+
			"User Migration Errors: %d\n"+
			"Ticket Migration Errors: %d\n",
			m.usersProcessed,
			m.newUsersCreated,
			m.ticketsProcessed,
			m.newTicketsCreated,
			m.ticketOrgsProcessed, len(m.data.SelectedOrgs),
			m.orgsNotInPsa,
			m.userMigrationErrors,
			m.ticketMigrationErrors)
	}

	mainView := lipgloss.NewStyle().
		Width(m.windowWidth).
		Height(m.verticalLeftForMainView).
		PaddingLeft(1).
		Render(s)

	views := []string{m.titleBar("Ticket Migration Utility"), mainView, m.viewportDivider(), m.viewport.View(), m.appFooter()}

	return lipgloss.JoinVertical(lipgloss.Top, views...)
}

func welcomeText() string {
	return fmt.Sprintf(`
This utility will copy all users and tickets from Zendesk to ConnectWise PSA.

Custom fields will be updated in both systems to reflect the migrationStatus of each item; if you run it again, it will only copy new items.

It is recommended to make your terminal as big as possible to see all output, as it will overflow horizontally in the below "Results" section. For full output, press %s to copy to clipboard.

If you exit in the middle of a migration, there may be incomplete tickets - %s

Press %s to select organizations and begin the migration. For more options, see the README.
`, textBlue("C"),
		textYellow("you will need to delete these before running the utility again."),
		textBlue("SPACE"))
}

func (c *Client) getTimeZone() (*time.Location, error) {
	ls := "UTC"
	if c.Cfg.TimeZone != "" {
		slog.Debug("time zone manually set in config", "timeZone", c.Cfg.TimeZone)
		ls = c.Cfg.TimeZone
	}

	loc, err := time.LoadLocation(ls)
	if err != nil {
		return nil, fmt.Errorf("converting time zone: %w", err)
	}

	return loc, nil
}

func (m *Model) copyToClipboard(s string) tea.Cmd {
	return func() tea.Msg {
		re := regexp.MustCompile(`\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])`)
		plaintext := re.ReplaceAllString(s, "")
		if err := clipboard.WriteAll(plaintext); err != nil {
			slog.Error("copying results to clipboard", "error", err)
			m.writeToOutput(badRedOutput("ERROR", "couldn't copy results to clipboard"), errOutput)
			return nil
		}
		slog.Debug("copied result to clipboard")
		m.writeToOutput(goodGreenOutput("SUCCESS", "copied results to clipboard"), createdOutput)
		return nil
	}
}
