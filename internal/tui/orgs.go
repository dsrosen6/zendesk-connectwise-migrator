package tui

import tea "github.com/charmbracelet/bubbletea"

type orgCheckerModel struct{}

func newOrgCheckerModel() orgCheckerModel {
	return orgCheckerModel{}
}

func (m orgCheckerModel) Init() tea.Cmd {
	return nil
}

func (m orgCheckerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m orgCheckerModel) View() string {
	return "something"
}
