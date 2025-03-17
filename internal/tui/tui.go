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
)

type Model struct {
	migrationClient *migration.Client
	currentModel    tea.Model
	quitting        bool
	ready           bool
	dimensions
	viewport      viewport.Model
	viewportTitle string
	viewportBody  string
}

type dimensions struct {
	windowWidth             int
	windowHeight            int
	verticalLeftForMainView int
}

type timeConversionDetails struct {
	startString string
	endString   string

	// fallback time strings, in case main entry is a blank string
	startFallback string
	endFallback   string
}

type switchModelMsg tea.Model

func NewModel(cx context.Context, mc *migration.Client) Model {
	ctx = cx

	spnr = spinner.New()
	spnr.Spinner = spinner.Ellipsis
	spnr.Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})

	mm := newMainMenuModel(mc)

	return Model{
		migrationClient: mc,
		currentModel:    mm,
		viewportTitle:   "Results",
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.currentModel.Init(),
		spnr.Tick,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		mainHeaderHeight := lipgloss.Height(m.appHeader())
		mainFooterHeight := lipgloss.Height(m.appFooter())
		viewportDvdrHeight := lipgloss.Height(m.viewportDivider())
		verticalMarginHeight := mainHeaderHeight + mainFooterHeight + viewportDvdrHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, (msg.Height-verticalMarginHeight)*2/3)

			m.verticalLeftForMainView = m.windowHeight - verticalMarginHeight - m.viewport.Height
			m.viewport.SetContent(m.viewportBody)
			m.ready = true

		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = (msg.Height - verticalMarginHeight) * 2 / 3
			m.viewport.SetContent(m.viewportBody)
		}

	case tea.KeyMsg:

		switch msg.String() {
		case "esc":
			m.quitting = true
			cmds = append(cmds, tea.Quit)
		case "ctrl+c":
			cmds = append(cmds, copyToClipboard(m.viewportBody))
		}

	case switchModelMsg:
		slog.Debug("got switchModelCmd", "model", msg)
		m.currentModel = msg
		cmds = append(cmds, m.currentModel.Init())

	case updateResultsMsg:
		slog.Debug("got updateViewportCmd")
		m.viewportTitle = msg.title
		m.viewportBody = msg.body
	}

	spnr, cmd = spnr.Update(msg)
	cmds = append(cmds, cmd)

	m.currentModel, cmd = m.currentModel.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport.SetContent(m.viewportBody)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if !m.ready {
		return runSpinner("Initializing...")
	}

	mainView := lipgloss.NewStyle().
		Width(m.windowWidth).
		Height(m.verticalLeftForMainView).
		Render(m.currentModel.View())

	return lipgloss.JoinVertical(lipgloss.Top, m.appHeader(), mainView, m.viewportDivider(), m.viewport.View(), m.appFooter())
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

func (m Model) appHeader() string {
	return m.titleBar("Ticket Migration Utility")
}

func (m Model) viewportDivider() string {
	return m.titleBar(m.viewportTitle)
}

func (m Model) titleBar(t string) string {
	titleBox := titleStyle().Render(t)

	titleBoxWidth := lipgloss.Width(titleBox)

	dividerLength := m.windowWidth - titleBoxWidth

	return lipgloss.JoinHorizontal(lipgloss.Center, titleBox, line(dividerLength))
}

func (m Model) appFooter() string {
	return m.titleBar("CTRL+C: Copy Results | ESC: Exit")
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
