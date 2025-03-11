package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"log/slog"
)

type mainMenuModel struct {
	form        *huh.Form
	currentMenu menuChoice
}

type menuChoice string

const (
	mainMenu      menuChoice = "mainMenu"
	checkOrgsMenu menuChoice = "checkOrgsMenu"
)

func newMainMenuModel() mainMenuModel {
	return mainMenuModel{
		form:        mainMenuForm(),
		currentMenu: mainMenu,
	}
}

func (mm mainMenuModel) Init() tea.Cmd {
	return mm.form.Init()
}

func (mm mainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return mm, tea.Quit
		}
	}

	form, cmd := mm.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		mm.form = f
	}

	if mm.form.State == huh.StateCompleted {
		choice := mm.form.Get("menuChoice")
		switch choice {
		case checkOrgsMenu:
			slog.Debug("switching to checkOrgsMenu")
			currentModel = newCheckOrgsModel()
			return currentModel, nil
		}
	}

	return mm, cmd
}

func (mm mainMenuModel) View() string {
	if mm.form.State == huh.StateCompleted {
		choice := mm.form.Get("menuChoice")
		return showStatus(fmt.Sprintf("You picked %s", choice))
	}
	return mm.form.View()
}

func mainMenuForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[menuChoice]().
				Title("Select an action to begin").
				Options(
					huh.NewOption("Check Orgs", checkOrgsMenu),
				).
				Key("menuChoice"),
		),
	)
}
