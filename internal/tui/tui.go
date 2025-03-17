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
	"strings"
	"time"
)

var (
	ctx  context.Context
	spnr spinner.Model

	titleStyle = func() lipgloss.Style {
		b := lipgloss.NormalBorder()
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}

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
	migrationClient *migration.Client
	currentModel    tea.Model
	migrationData   *migrationData
	viewport        viewPort
	quitting        bool
}

type migrationData struct {
	readyOrgs          []*orgMigrationDetails
	orgsToMigrateUsers []*orgMigrationDetails
}

type orgMigrationDetails struct {
	zendeskOrg *zendesk.Organization
	psaOrg     *psa.Company

	usersToMigrate []*userMigrationDetails
	// TODO: ticketsToMigrate []*ticketMigrationDetails

	tag        *tagDetails
	hasTickets bool

	ready bool
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

func NewModel(cx context.Context, mc *migration.Client) *Model {
	ctx = cx

	spnr = spinner.New()
	spnr.Spinner = spinner.Ellipsis
	spnr.Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})

	data := &migrationData{}
	mm := newMainMenuModel(mc, data)

	return &Model{
		migrationClient: mc,
		currentModel:    mm,
		migrationData:   data,
		viewport:        viewPort{title: "Results", show: false},
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.currentModel.Init(),
		spnr.Tick,
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "r":
			cmds = append(cmds,
				switchModel(newMainMenuModel(m.migrationClient, m.migrationData)),
				toggleViewport(false))
		}

	case switchModelMsg:
		slog.Debug("received new model via switchModelMsg", "model", msg)
		m.currentModel = msg
		cmds = append(cmds, m.currentModel.Init())

	case updateResultsMsg:
		slog.Debug("received updated viewport content via updateResultsMsg")
		m.viewport.body = msg.body

	case toggleViewportMsg:
		m.viewport.show = msg.on
		return m, calculateDimensions(windowWidth, windowHeight, m.viewport)

	case sendOrgsMsg:
		slog.Debug("received orgs via sendOrgsMsg")
		m.migrationData.readyOrgs = msg
	}

	spnr, cmd = spnr.Update(msg)
	cmds = append(cmds, cmd)

	m.currentModel, cmd = m.currentModel.Update(msg)
	cmds = append(cmds, cmd)

	if m.viewport.show {
		m.viewport.model.SetContent(m.viewport.body)
		m.viewport.model, cmd = m.viewport.model.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
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

	views := []string{appHeader(), mainView}

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

func appHeader() string {
	return titleBar("Ticket Migration Utility")
}

func viewportDivider(v viewPort) string {
	return titleBar(v.title)
}

func appFooter() string {
	return titleBar("C: Copy Results | R: Main Menu | ESC: Exit")
}

func titleBar(t string) string {
	titleBox := titleStyle().Render(t)

	titleBoxWidth := lipgloss.Width(titleBox)

	dividerLength := windowWidth - titleBoxWidth

	return lipgloss.JoinHorizontal(lipgloss.Center, titleBox, line(dividerLength))
}

func line(w int) string {
	line := strings.Repeat("─", maxRepeats(0, w))
	return line
}

func maxRepeats(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TODO: add an on screen instruction for this
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
	return func() tea.Msg {
		windowWidth = w
		windowHeight = h
		mainHeaderHeight = lipgloss.Height(appHeader())
		mainFooterHeight = lipgloss.Height(appFooter())
		viewportDvdrHeight = lipgloss.Height(viewportDivider(v))
		return calculateDimensionsMsg{}
	}
}
