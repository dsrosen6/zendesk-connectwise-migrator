package tui

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"log/slog"
)

var (
	ctx  context.Context
	spnr spinner.Model
)

type Model struct {
	migrationClient *migration.Client
	currentModel    tea.Model
	quitting        bool
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

	return m.currentModel.View()
}

func switchModel(sm tea.Model) tea.Cmd {
	return func() tea.Msg {
		return switchModelMsg(sm)
	}
}

func runSpinner(text string) string {
	return fmt.Sprintf("%s %s\n", spnr.View(), text)
}
