package tui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"log/slog"
)

var (
	ctx context.Context
)

type Model struct {
	migrationClient *migration.Client
	currentModel    tea.Model
}

type switchModelMsg tea.Model

func NewModel(cx context.Context, mc *migration.Client) Model {
	ctx = cx

	mm := newMainMenuModel(mc)

	return Model{
		migrationClient: mc,
		currentModel:    mm,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.currentModel.Init(),
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
		case "ctrl+c":
			cmds = append(cmds, tea.Quit)
		}

	case switchModelMsg:
		slog.Debug("got switchModelCmd", "model", msg)
		m.currentModel = msg
		cmds = append(cmds, m.currentModel.Init())
	}

	m.currentModel, cmd = m.currentModel.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.currentModel.View()
}

func switchModel(sm tea.Model) tea.Cmd {
	return func() tea.Msg {
		return switchModelMsg(sm)
	}
}
