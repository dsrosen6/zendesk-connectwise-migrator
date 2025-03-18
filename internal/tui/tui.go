package tui

import (
	"context"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
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

	menuTabs = []menuTab{tabMainPage, tabOrgs, tabUsers}
)

type menuTab string

const (
	tabMainPage menuTab = "M | Main Page"
	tabOrgs     menuTab = "O | Organizations"
	tabUsers    menuTab = "U | Users"
)

type RootModel struct {
	client       *migration.Client
	submodels    *submodels
	currentModel tea.Model
	data         *migrationData
	activeTab    menuTab
	viewport     viewPort
	quitting     bool
}

type submodels struct {
	mainPage      tea.Model
	orgMigration  tea.Model
	userMigration tea.Model
}

type migrationData struct {
	orgs []*orgMigrationDetails
}

type orgMigrationDetails struct {
	zendeskOrg   *zendesk.Organization
	psaOrg       *psa.Company
	orgMigErrors []error

	tag        *tagDetails
	hasTickets bool

	readyUsers      bool
	userMigSelected bool
	usersToMigrate  []*userMigrationDetails
	userMigErrors   []error
	// TODO: ticketsToMigrate []*ticketMigrationDetails
}

type userMigrationDetails struct {
	zendeskUser *zendesk.User
	psaContact  *psa.Contact
	ready       bool
}

type viewPort struct {
	model viewport.Model
	title string
	body  string
	show  bool
	ready bool
}

type timeConversionDetails struct {
	startString string
	endString   string

	// fallback time strings, in case main entry is a blank string
	startFallback string
	endFallback   string
}

type switchModelMsg tea.Model

type sendOrgsMsg []*orgMigrationDetails

type calculateDimensionsMsg struct{}

func sendOrgsCmd(orgs []*orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		return sendOrgsMsg(orgs)
	}
}

func NewModel(cx context.Context, client *migration.Client) *RootModel {
	ctx = cx

	spnr = spinner.New()
	spnr.Spinner = spinner.Ellipsis
	spnr.Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})

	data := &migrationData{}
	mm := newMainMenuModel(client, data)

	sm := &submodels{
		mainPage:      newMainMenuModel(client, data),
		orgMigration:  newOrgMigrationModel(client, data),
		userMigration: newUserMigrationModel(client, data),
	}

	return &RootModel{
		client:       client,
		submodels:    sm,
		currentModel: mm,
		data:         data,
		activeTab:    tabMainPage,
		viewport:     viewPort{title: "Results", show: false},
	}
}

func (m *RootModel) Init() tea.Cmd {
	return tea.Batch(
		m.currentModel.Init(),
		spnr.Tick,
	)
}

func (m *RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, calculateDimensions(msg.Width, msg.Height, m.viewport)

	case calculateDimensionsMsg:
		if !m.viewport.ready {
			m.viewport.model = viewport.New(windowWidth, (windowHeight-verticalMarginHeight)*2/3)
			m.viewport.model.SetContent(m.viewport.body)
			m.viewport.ready = true
		} else {
			m.viewport.model.Width = windowWidth
			m.viewport.model.Height = (windowHeight - verticalMarginHeight) * 2 / 3
			m.viewport.model.SetContent(m.viewport.body)
		}

		if m.viewport.show {
			verticalMarginHeight = mainHeaderHeight + mainFooterHeight + viewportDvdrHeight
			verticalLeftForMainView = windowHeight - verticalMarginHeight - m.viewport.model.Height
		} else {
			verticalMarginHeight = mainHeaderHeight + mainFooterHeight
			verticalLeftForMainView = windowHeight - verticalMarginHeight
		}

	case tea.KeyMsg:

		switch msg.String() {
		case "esc":
			m.quitting = true
			cmds = append(cmds, tea.Quit)
		case "c":
			cmds = append(cmds, copyToClipboard(m.viewport.body))
		case "m":
			if m.activeTab != tabMainPage {
				m.activeTab = tabMainPage
				cmds = append(cmds,
					switchModel(m.submodels.mainPage),
					toggleViewport(false))
			}
		case "o":
			// TODO: figure out why resize happens twice on double o press
			if m.activeTab != tabOrgs {
				m.activeTab = tabOrgs
				cmds = append(cmds,
					switchModel(m.submodels.orgMigration),
					toggleViewport(true))
			}
		case "u":
			if m.activeTab != tabUsers {
				m.activeTab = tabUsers
				cmds = append(cmds,
					switchModel(m.submodels.userMigration),
					toggleViewport(false))
			}
		}

	case switchModelMsg:
		slog.Debug("received new model via switchModelMsg", "model", msg)
		m.currentModel = msg
		cmds = append(cmds, m.currentModel.Init())

	case updateResultsMsg:
		slog.Debug("received updated viewport content via updateResultsMsg")
		m.viewport.body = msg.body

	case toggleViewportMsg:
		slog.Debug("received toggle viewport msg", "on", msg.on)
		m.viewport.show = msg.on
		return m, calculateDimensions(windowWidth, windowHeight, m.viewport)

	case sendOrgsMsg:
		slog.Debug("received orgs via sendOrgsMsg")
		m.data.orgs = msg
		return m, switchUserMigStatus(pickingOrgs)
	}

	spnr, cmd = spnr.Update(msg)
	cmds = append(cmds, cmd)

	m.submodels.mainPage, cmd = m.submodels.mainPage.Update(msg)
	cmds = append(cmds, cmd)

	m.submodels.orgMigration, cmd = m.submodels.orgMigration.Update(msg)
	cmds = append(cmds, cmd)

	m.submodels.userMigration, cmd = m.submodels.userMigration.Update(msg)
	cmds = append(cmds, cmd)

	if m.viewport.show {
		m.viewport.model.SetContent(m.viewport.body)
		m.viewport.model, cmd = m.viewport.model.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *RootModel) View() string {
	if m.quitting {
		return ""
	}

	if !m.viewport.ready {
		return runSpinner("Initializing...")
	}

	mainView := lipgloss.NewStyle().
		Width(windowWidth).
		Height(verticalLeftForMainView).
		PaddingLeft(1).
		Render(m.currentModel.View())

	views := []string{menuBar(menuTabs, m.activeTab), mainView}

	if m.viewport.show {
		views = append(views, viewportDivider(m.viewport), m.viewport.model.View(), appFooter())
	} else {
		views = append(views, appFooter())
	}

	return lipgloss.JoinVertical(lipgloss.Top, views...)
}

func switchModel(sm tea.Model) tea.Cmd {
	return func() tea.Msg {
		return switchModelMsg(sm)
	}
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

func copyToClipboard(s string) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.WriteAll(s); err != nil {
			slog.Error("copying result to clipboard", "error", err)
			return nil
		}
		slog.Debug("copied result to clipboard")
		return nil
	}
}

func calculateDimensions(w, h int, v viewPort) tea.Cmd {
	dummyMenuTabs := []menuTab{tabMainPage}
	return func() tea.Msg {
		windowWidth = w
		windowHeight = h
		mainHeaderHeight = lipgloss.Height(menuBar(dummyMenuTabs, dummyMenuTabs[0]))
		mainFooterHeight = lipgloss.Height(appFooter())
		viewportDvdrHeight = lipgloss.Height(viewportDivider(v))
		return calculateDimensionsMsg{}
	}
}
