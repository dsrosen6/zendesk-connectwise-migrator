package tui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	currentModel tea.Model = newMainMenuModel()
	ctx          context.Context
)

type Model struct {
	currentModel tea.Model
}

func NewModel(cx context.Context) Model {
	ctx = cx
	return Model{
		currentModel: currentModel,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.currentModel.Update(msg)
}

func (m Model) View() string {
	return m.currentModel.View()
}

// This will eventually be a spinner.
func showStatus(s string) string {
	return s
}
