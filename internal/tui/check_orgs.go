package tui

import tea "github.com/charmbracelet/bubbletea"

type checkOrgsModel struct {
}

func newCheckOrgsModel() checkOrgsModel {
	return checkOrgsModel{}
}

func (co checkOrgsModel) Init() tea.Cmd {
	return nil
}

func (co checkOrgsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return co, tea.Quit
		}
	}
	return co, nil
}

func (co checkOrgsModel) View() string {
	return "We checkin orgs"
}
