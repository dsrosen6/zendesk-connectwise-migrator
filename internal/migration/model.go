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
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

var (
	spnr spinner.Model

	// App Dimensions
	windowWidth             int
	windowHeight            int
	mainHeaderHeight        int
	mainFooterHeight        int
	viewportDvdrHeight      int
	verticalMarginHeight    int
	verticalLeftForMainView int
)

type Model struct {
	ctx             context.Context
	client          *Client
	data            *Data
	form            *huh.Form
	status          status
	timeZone        *time.Location
	allOrgsSelected bool
	viewport        viewport.Model
	statistics
	viewState
	scrollManagement
}

type Data struct {
	AllOrgs      map[string]*orgMigrationDetails
	UsersInPsa   map[string]*userMigrationDetails
	TicketsInPsa map[string]int

	PsaInfo        PsaInfo
	Tags           []tagDetails
	SelectedOrgs   []*orgMigrationDetails
	UsersToMigrate map[string]*userMigrationDetails

	Output strings.Builder
}

type outputLevel string

const (
	noActionOutput outputLevel = "noActionOutput"
	createdOutput  outputLevel = "createdActionOutput"
	warnOutput     outputLevel = "warnActionOutput"
	errOutput      outputLevel = "errorActionOutput"
)

type status string

const (
	awaitingStart      status = "Awaiting Start"
	gettingTags        status = "Getting Tags from Config"
	gettingZendeskOrgs status = "Getting Zendesk Organizations"
	comparingOrgs      status = "Checking for Organization Matches"
	initOrgForm        status = "Initializing Form"
	pickingOrgs        status = "Selecting Organizations"
	gettingUsers       status = "Getting Users"
	migratingUsers     status = "Migrating Users"
	gettingPsaTickets  status = "Getting PSA Tickets"
	migratingTickets   status = "Migrating Tickets"
	done               status = "Done"
)

type switchStatusMsg status

func switchStatus(s status) tea.Cmd {
	return func() tea.Msg { return switchStatusMsg(s) }
}

type statistics struct {
	orgsChecked         int
	orgsNotInPsa        int
	orgsMigrated        int
	orgsCheckedForUsers int
	usersToCheck        int
	usersMigrated       int
	usersSkipped        int
	ticketsToMigrate    int
	ticketsMigrated     int
	ticketOrgsMigrated  int
	totalErrors         int
}

type PsaInfo struct {
	Board                  *psa.Board
	StatusOpen             *psa.Status
	StatusClosed           *psa.Status
	ZendeskTicketIdField   *psa.CustomField
	ZendeskClosedDateField *psa.CustomField
}

type timeConversionDetails struct {
	startString string
	endString   string

	// fallback time strings, in case main entry is a blank string
	startFallback string
	endFallback   string
}

type viewState struct {
	ready    bool
	quitting bool
}

type scrollManagement struct {
	scrollOverride  bool
	scrollCountDown bool
}

func newModel(ctx context.Context, client *Client) (*Model, error) {
	spnr = spinner.New()
	spnr.Spinner = spinner.Ellipsis
	spnr.Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})

	data := &Data{
		AllOrgs:        make(map[string]*orgMigrationDetails),
		UsersInPsa:     make(map[string]*userMigrationDetails),
		TicketsInPsa:   make(map[string]int),
		UsersToMigrate: make(map[string]*userMigrationDetails),
	}

	if client.Cfg.TicketLimit > 0 {
		slog.Info("ticket test limit in config", "limit", client.Cfg.TicketLimit)
	}

	data.PsaInfo = PsaInfo{
		Board:                  &psa.Board{Id: client.Cfg.Connectwise.DestinationBoardId},
		StatusOpen:             &psa.Status{Id: client.Cfg.Connectwise.OpenStatusId},
		StatusClosed:           &psa.Status{Id: client.Cfg.Connectwise.ClosedStatusId},
		ZendeskTicketIdField:   &psa.CustomField{Id: client.Cfg.Connectwise.FieldIds.ZendeskTicketId},
		ZendeskClosedDateField: &psa.CustomField{Id: client.Cfg.Connectwise.FieldIds.ZendeskClosedDate},
	}

	loc, err := client.getTimeZone()
	if err != nil {
		slog.Error("getting time zone", "error", err)
		return nil, fmt.Errorf("getting time zone: %w", err)
	}

	slog.Info("time zone set", "timeZone", loc.String())
	slog.Info("migrate open tickets set to", "value", client.Cfg.MigrateOpenTickets)
	slog.Info("output levels in config", "levels", client.Cfg.OutputLevels)

	return &Model{
		ctx:      ctx,
		client:   client,
		data:     data,
		status:   awaitingStart,
		timeZone: loc,
	}, nil
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

func (m *Model) Init() tea.Cmd {
	return spnr.Tick
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
		case "esc":
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
		m.status = status(msg)
		switch status(msg) {
		case gettingTags:
			slog.Debug("getting tags from config")
			return m, m.getTagDetails()
		case gettingZendeskOrgs:
			slog.Debug("getting zendesk orgs")
			m.statistics = statistics{}
			return m, m.getOrgs()
		case comparingOrgs:
			var checkOrgCmds []tea.Cmd
			for _, org := range m.data.AllOrgs {
				checkOrgCmds = append(checkOrgCmds, m.checkOrg(org))
			}

			return m, tea.Sequence(checkOrgCmds...)
		case initOrgForm:
			m.form = m.orgSelectionForm()
			cmds = append(cmds, m.form.Init(), switchStatus(pickingOrgs))
			return m, tea.Sequence(cmds...)
		case gettingUsers:
			m.data.UsersToMigrate = make(map[string]*userMigrationDetails)
			for _, org := range m.data.SelectedOrgs {
				cmds = append(cmds, m.getUsersToMigrate(org))
			}

			return m, tea.Sequence(cmds...)
		case migratingUsers:
			for _, user := range m.data.UsersToMigrate {
				cmds = append(cmds, m.migrateUser(user))
			}

			return m, tea.Sequence(cmds...)
		case gettingPsaTickets:
			return m, m.getAlreadyMigrated()
		case migratingTickets:
			for _, org := range m.data.SelectedOrgs {
				cmds = append(cmds, m.runTicketMigration(org))
			}

			return m, tea.Sequence(cmds...)
		}
	}

	spnr, cmd = spnr.Update(msg)
	cmds = append(cmds, cmd)

	switch m.status {
	case comparingOrgs:
		if len(m.data.AllOrgs) == m.orgsChecked {
			cmds = append(cmds, switchStatus(initOrgForm))
		}

	case pickingOrgs:
		form, cmd := m.form.Update(msg)
		cmds = append(cmds, cmd)

		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		if m.form.State == huh.StateCompleted {
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

				cmds = append(cmds, switchStatus(gettingUsers))
			}
		}

	case gettingUsers:
		if len(m.data.SelectedOrgs) == m.orgsCheckedForUsers {
			cmds = append(cmds, switchStatus(migratingUsers))
		}

	case migratingUsers:
		if len(m.data.UsersToMigrate) == m.usersMigrated+m.usersSkipped {
			slog.Info("all users migrated, beginning ticket migration")
			cmds = append(cmds, switchStatus(gettingPsaTickets))
		}

	case migratingTickets:
		if len(m.data.SelectedOrgs) == m.ticketOrgsMigrated {
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
		return runSpinner("Initializing")
	}

	var s string
	switch m.status {
	case awaitingStart:
		s += "Press the SPACE key to begin"
	case pickingOrgs:
		s += m.form.View()
	case done:
		s += "Migration complete"
	default:
		s += runSpinner(string(m.status))
	}

	if m.status != awaitingStart && m.status != pickingOrgs {
		s += fmt.Sprintf("\n\nUsers (Processed/Total): %d/%d\n"+
			"Tickets (Migrated/Total): %d/%d\n"+
			"Orgs Complete: %d\n"+
			"Errors: %d\n",
			m.usersMigrated+m.usersSkipped, m.usersToCheck,
			m.ticketsMigrated, m.ticketsToMigrate,
			m.ticketOrgsMigrated,
			m.totalErrors)
	}

	mainView := lipgloss.NewStyle().
		Width(windowWidth).
		Height(verticalLeftForMainView).
		PaddingLeft(1).
		Render(s)

	views := []string{titleBar("Ticket Migration Utility"), mainView, viewportDivider(), m.viewport.View(), appFooter()}

	return lipgloss.JoinVertical(lipgloss.Top, views...)
}

func runSpinner(text string) string {
	return fmt.Sprintf("%s%s", text, spnr.View())
}

func (m *Model) copyToClipboard(s string) tea.Cmd {
	return func() tea.Msg {
		re := regexp.MustCompile(`\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])`)
		plaintext := re.ReplaceAllString(s, "")
		if err := clipboard.WriteAll(plaintext); err != nil {
			slog.Error("copying results to clipboard", "error", err)
			m.writeToOutput(badRedOutput("ERROR", fmt.Errorf("couldn't copy results to clipboard: %w", err)), errOutput)
			return nil
		}
		slog.Debug("copied result to clipboard")
		m.writeToOutput(goodGreenOutput("SUCCESS", "copied results to clipboard"), createdOutput)
		return nil
	}
}

func (m *Model) writeToOutput(s string, level outputLevel) {
	switch level {
	case noActionOutput:
		if m.client.Cfg.OutputLevels.NoAction {
			m.data.Output.WriteString(s)
		}
	case createdOutput:
		if m.client.Cfg.OutputLevels.Created {
			m.data.Output.WriteString(s)
		}
	case warnOutput:
		if m.client.Cfg.OutputLevels.Warn {
			m.data.Output.WriteString(s)
		}
	case errOutput:
		if m.client.Cfg.OutputLevels.Error {
			m.data.Output.WriteString(s)
		}
	}
}

func (m *Model) setAutoScrollBehavior() {
	if m.viewport.AtBottom() {
		m.scrollOverride = false
	}

	if !m.scrollOverride {
		m.viewport.GotoBottom()
	}
}
