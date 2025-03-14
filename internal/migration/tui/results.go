package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type updateViewportMsg struct {
	title string
	body  string
}

func sendResultsCmd(title, body string) tea.Cmd {
	return func() tea.Msg {
		return updateViewportMsg{
			title: title,
			body:  body,
		}
	}
}
