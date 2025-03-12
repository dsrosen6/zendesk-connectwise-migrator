package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	model   tea.Model
	mmModel tea.Model
	ocModel tea.Model
}

type switchModelMsg tea.Model

func NewModel() Model {
	mm := newMainMenuModel()
	oc := newOrgCheckerModel()
	return Model{
		model:   mm,
		mmModel: mm,
		ocModel: oc,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.mmModel.Init(),
		m.ocModel.Init(),
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
		m.model = msg
	}

	m.model, cmd = m.model.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.model.View()
}

func switchModel(sm tea.Model) tea.Cmd {
	return func() tea.Msg {
		return switchModelMsg(sm)
	}
}
