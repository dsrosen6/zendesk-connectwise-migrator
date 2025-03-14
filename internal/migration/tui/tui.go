package tui

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"log/slog"
	"time"
)

var (
	ctx  context.Context
	spnr spinner.Model

	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()
)

type Model struct {
	migrationClient *migration.Client
	currentModel    tea.Model
	quitting        bool
	ready           bool
	dimensions
}

type dimensions struct {
	width  int
	height int
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
	spnr.Spinner = spinner.Line
	spnr.Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})

	mm := newMainMenuModel(mc)

	return Model{
		migrationClient: mc,
		currentModel:    mm,
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
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tea.KeyMsg:

		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			cmds = append(cmds, tea.Quit)
		}

	case switchModelMsg:
		slog.Debug("got switchModelCmd", "model", msg)
		m.currentModel = msg
		cmds = append(cmds, m.currentModel.Init())
	}

	spnr, cmd = spnr.Update(msg)
	cmds = append(cmds, cmd)

	m.currentModel, cmd = m.currentModel.Update(msg)
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

	header := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.width).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		Render("Ticket Migration Utility")
	footer := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.width).
		Render("")
	content := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height-lipgloss.Height(header)-lipgloss.Height(footer)).
		Align(lipgloss.Center, lipgloss.Top).
		Render(m.currentModel.View())

	return lipgloss.JoinVertical(lipgloss.Top, header, content, footer)
}

func switchModel(sm tea.Model) tea.Cmd {
	return func() tea.Msg {
		return switchModelMsg(sm)
	}
}

func runSpinner(text string) string {
	return fmt.Sprintf("%s %s\n", spnr.View(), text)
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
