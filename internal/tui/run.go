package tui

import tea "github.com/charmbracelet/bubbletea"

import "veltryx/internal/monitoring"

func Run(monitor *monitoring.Monitor) error {
	p := tea.NewProgram(newModel(monitor), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
