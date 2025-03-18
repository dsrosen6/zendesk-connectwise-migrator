package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
)

type mainMenuModel struct {
	migrationClient *migration.Client
	migrationData   *MigrationData
}

const (
	mainMenuChoice = "mainMenuChoice"
	orgMigrator    = "orgMigrator"
	userMigrator   = "userMigrator"
)

func newMainMenuModel(mc *migration.Client, data *MigrationData) *mainMenuModel {
	return &mainMenuModel{
		migrationClient: mc,
		migrationData:   data,
	}
}

func (m *mainMenuModel) Init() tea.Cmd {
	return nil
}

func (m *mainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *mainMenuModel) View() string {
	return "Main page"
}
