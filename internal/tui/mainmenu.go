package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

type mainMenuModel struct {
	form *huh.Form
}

func newMainMenuModel() mainMenuModel {
	f := mainMenuForm()
	return mainMenuModel{form: f}
}

func (m mainMenuModel) Init() tea.Cmd {
	return tea.Sequence(m.form.Init())
}

func (m mainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
	)

	form, cmd := m.form.Update(msg)
	cmds = append(cmds, cmd)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		choice := m.form.GetString("mainMenuChoice")
		switch choice {
		case "orgChecker":
			cmds = append(cmds, switchModel(orgCheckerModel{}))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m mainMenuModel) View() string {
	return m.form.View()
}

func mainMenuForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Pick an option").
				Options(
					huh.NewOption("Org Checker", "orgChecker"),
				).
				Key("mainMenuChoice"),
		))
}
