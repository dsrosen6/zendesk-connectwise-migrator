package migration

import tea "github.com/charmbracelet/bubbletea"

type model struct {
}

func initialModel() model {
	return model{}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	s := "Hello"
	return s
}
