package migration

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

func CustomHuhTheme() *huh.Theme {
	t := huh.ThemeBase()
	t.Focused.Base = lipgloss.NewStyle()
	t.Blurred.Base = t.Focused.Base
	return t
}

func quitKeyMap() key.Binding {
	return key.NewBinding(key.WithKeys("esc", "ctrl+c"))
}

func customKeyMap() *huh.KeyMap {
	k := huh.NewDefaultKeyMap()
	k.Quit = quitKeyMap()

	return k
}
