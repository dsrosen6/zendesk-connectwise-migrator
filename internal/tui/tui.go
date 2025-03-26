package tui

import (
	"context"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"log/slog"
	"strings"
	"time"
)

var (
	ctx  context.Context
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

type RootModel struct {
	client *migration.Client
	data   *MigrationData
	form   *huh.Form
	statistics
	allOrgsSelected bool
	submodels       *submodels
	viewport        viewport.Model
	viewState
	scrollManagement
	status   status
	timeZone *time.Location
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
	ticketOrgsToMigrate int
	ticketOrgsMigrated  int
	totalErrors         int
}

type submodels struct {
	mainPage        tea.Model
	orgMigration    tea.Model
	userMigration   tea.Model
	ticketMigration tea.Model
}

type MigrationData struct {
	AllOrgs      map[string]*orgMigrationDetails
	UsersInPsa   map[string]*userMigrationDetails
	TicketsInPsa map[string]int

	PsaInfo        PsaInfo
	Tags           []tagDetails
	SelectedOrgs   []*orgMigrationDetails
	UsersToMigrate map[string]*userMigrationDetails

	Output strings.Builder
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

type switchStatusMsg status

func switchStatus(s status) tea.Cmd {
	return func() tea.Msg { return switchStatusMsg(s) }
}

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

func NewModel(cx context.Context, client *migration.Client) (*RootModel, error) {
	ctx = cx

	spnr = spinner.New()
	spnr.Spinner = spinner.Ellipsis
	spnr.Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})

	data := &MigrationData{
		AllOrgs:        make(map[string]*orgMigrationDetails),
		UsersInPsa:     make(map[string]*userMigrationDetails),
		TicketsInPsa:   make(map[string]int),
		UsersToMigrate: make(map[string]*userMigrationDetails),
	}

	if data.TicketsInPsa == nil {
		data.TicketsInPsa = make(map[string]int)
	}

	if data.UsersInPsa == nil {
		data.UsersInPsa = make(map[string]*userMigrationDetails)
	}

	if client.Cfg.TestLimit > 0 {
		slog.Info("ticket test limit in config", "limit", client.Cfg.TestLimit)
	}

	data.PsaInfo = PsaInfo{
		Board:                  &psa.Board{Id: client.Cfg.Connectwise.DestinationBoardId},
		StatusOpen:             &psa.Status{Id: client.Cfg.Connectwise.OpenStatusId},
		StatusClosed:           &psa.Status{Id: client.Cfg.Connectwise.ClosedStatusId},
		ZendeskTicketIdField:   &psa.CustomField{Id: client.Cfg.Connectwise.FieldIds.ZendeskTicketId},
		ZendeskClosedDateField: &psa.CustomField{Id: client.Cfg.Connectwise.FieldIds.ZendeskClosedDate},
	}

	ls := "UTC"
	if client.Cfg.TimeZone != "" {
		slog.Debug("time zone manually set in config", "timeZone", client.Cfg.TimeZone)
		ls = client.Cfg.TimeZone
	}

	location, err := time.LoadLocation(ls)
	if err != nil {
		return nil, fmt.Errorf("converting time zone: %w", err)
	}
	slog.Info("setting time zone", "timeZone", ls)

	slog.Info("migrate open tickets set to", "value", client.Cfg.MigrateOpenTickets)

	return &RootModel{
		client:   client,
		data:     data,
		status:   awaitingStart,
		timeZone: location,
	}, nil
}

func (m *RootModel) Init() tea.Cmd {
	return spnr.Tick
}

func (m *RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

			return m, tea.Sequence(cmds...) // TODO: switch to batch when ready for speed
		case migratingUsers:
			for _, user := range m.data.UsersToMigrate {
				cmds = append(cmds, m.migrateUser(user))
			}

			return m, tea.Sequence(cmds...) // TODO: switch to batch when ready for speed
		case gettingPsaTickets:
			return m, m.getAlreadyMigrated()
		case migratingTickets:
			for _, org := range m.data.SelectedOrgs {
				cmds = append(cmds, m.runTicketMigration(org))
			}

			return m, tea.Sequence(cmds...) // TODO: switch to batch when ready for speed
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

func (m *RootModel) View() string {
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

	//if m.status != awaitingStart && m.status != pickingOrgs {
	//	s += fmt.Sprintf("\n\nOrgs With Tickets: %d\n"+
	//		"Total Users Processed: %d/%d\n"+
	//		"Tickets Migrated: %d/%d\n",
	//		m.orgsWithTickets, m.totalUsersChecked, m.totalUsersToCheck, m.ticketsMigrated, m.ticketsToMigrate)
	//}

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

func convertDateStringsToTimeTime(details *timeConversionDetails) (time.Time, time.Time, error) {
	var startDate, endDate time.Time
	var err error

	start := details.startString
	if start == "" {
		start = details.startFallback
	}

	end := details.endString
	if end == "" {
		end = details.endFallback
	}

	if start != "" {
		startDate, err = migration.ConvertStringToTime(start)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("converting start date string to time.Time: %w", err)
		}
	}

	if end != "" {
		endDate, err = migration.ConvertStringToTime(end)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("converting end date string to time.Time: %w", err)
		}
	}

	return startDate, endDate, nil
}

func (m *RootModel) copyToClipboard(s string) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.WriteAll(s); err != nil {
			slog.Error("copying results to clipboard", "error", err)
			m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("couldn't copy results to clipboard: %w", err)))
			return nil
		}
		slog.Debug("copied result to clipboard")
		m.data.writeToOutput(goodGreenOutput("SUCCESS", "copied results to clipboard"))
		return nil
	}
}

func (m *RootModel) calculateDimensions(w, h int) tea.Cmd {
	return func() tea.Msg {
		windowWidth = w
		windowHeight = h
		mainHeaderHeight = lipgloss.Height(titleBar("Ticket Migration Utility"))
		mainFooterHeight = lipgloss.Height(appFooter())
		viewportDvdrHeight = lipgloss.Height(viewportDivider())
		verticalMarginHeight = mainHeaderHeight + mainFooterHeight + viewportDvdrHeight
		viewportHeight := (windowHeight - verticalMarginHeight) * 1 / 2
		verticalLeftForMainView = windowHeight - verticalMarginHeight - viewportHeight
		slog.Debug("got calculateDimensionsMsg")

		if !m.ready {
			m.viewport = viewport.New(windowWidth, viewportHeight)
		} else {
			m.viewport.Width = windowWidth
			m.viewport.Height = viewportHeight
		}

		m.viewport.SetContent(m.data.Output.String())
		m.setAutoScrollBehavior()
		slog.Debug("setting ready to true")
		m.ready = true

		return nil
	}
}

func (d *MigrationData) writeToOutput(s string) {
	d.Output.WriteString(s)
}

func (m *RootModel) setAutoScrollBehavior() {
	if m.viewport.AtBottom() {
		m.scrollOverride = false
	}

	if !m.scrollOverride {
		m.viewport.GotoBottom()
	}
}
