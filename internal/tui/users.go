package tui

import tea "github.com/charmbracelet/bubbletea"

type userMigrationModel struct{}

func newUserMigrationModel() userMigrationModel {
	return userMigrationModel{}
}

func (m userMigrationModel) Init() tea.Cmd {
	return nil
}

func (m userMigrationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m userMigrationModel) View() string {
	return "something"
}
