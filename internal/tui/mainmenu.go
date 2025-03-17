package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"log/slog"
)

type mainMenuModel struct {
	migrationClient *migration.Client
	form            *huh.Form
}

const (
	mainMenuChoice = "mainMenuChoice"
	orgMigrator    = "orgMigrator"
	userMigrator   = "userMigrator"
)

func newMainMenuModel(mc *migration.Client) *mainMenuModel {
	f := mainMenuForm()
	return &mainMenuModel{
		migrationClient: mc,
		form:            f}
}

func (m *mainMenuModel) Init() tea.Cmd {
	return tea.Sequence(m.form.Init())
}

func (m *mainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
	)

	form, cmd := m.form.Update(msg)
	cmds = append(cmds, cmd)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		choice := m.form.GetString(mainMenuChoice)
		switch choice {
		case orgMigrator:
			slog.Debug("switching to org checker model")
			cmds = append(cmds, switchModel(newOrgCheckerModel(m.migrationClient)))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *mainMenuModel) View() string {
	return m.form.View()
}

func mainMenuForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Options(
					huh.NewOption("Organizations", orgMigrator),
					huh.NewOption("Users", userMigrator),
				).
				Key(mainMenuChoice),
		)).WithShowHelp(false).WithTheme(migration.CustomHuhTheme())
}
