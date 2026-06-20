package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
	Quit    key.Binding
	Tab     key.Binding
	Reset   key.Binding
	Logs    key.Binding
	Clients key.Binding
	Help    key.Binding
	Up      key.Binding
	Down    key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quitter")),
		Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "changer vue")),
		Reset:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reset session")),
		Logs:    key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "focus logs")),
		Clients: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "focus clients")),
		Help:    key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "aide")),
		Up:      key.NewBinding(key.WithKeys("up", "k", "pgup"), key.WithHelp("up", "monter")),
		Down:    key.NewBinding(key.WithKeys("down", "j", "pgdown"), key.WithHelp("down", "descendre")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Logs, k.Clients, k.Reset, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.Logs, k.Clients},
		{k.Up, k.Down, k.Reset},
		{k.Help, k.Quit},
	}
}

func newHelpModel() help.Model {
	h := help.New()
	h.ShowAll = false
	return h
}
