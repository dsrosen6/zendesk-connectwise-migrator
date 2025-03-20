package tui

import tea "github.com/charmbracelet/bubbletea"

type ticketMigrationModel struct{}

func newTicketMigrationModel() ticketMigrationModel {
	return ticketMigrationModel{}
}

func (m ticketMigrationModel) Init() tea.Cmd {
	return nil
}

func (m ticketMigrationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m ticketMigrationModel) View() string {
	return "something"
}
