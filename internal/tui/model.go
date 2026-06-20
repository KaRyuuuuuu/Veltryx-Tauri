package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"veltryx/internal/monitoring"
	"veltryx/internal/tui/panels"
)

type viewMode int

const (
	viewOverview viewMode = iota
	viewLogs
	viewClients
)

type snapshotMsg struct {
	snapshot monitoring.Snapshot
}

type model struct {
	monitor *monitoring.Monitor
	styles  styles
	keys    keyMap
	help    help.Model

	width       int
	height      int
	view        viewMode
	showHelp    bool
	last        monitoring.Snapshot
	logsView    viewport.Model
	clientsView viewport.Model
}

func newModel(monitor *monitoring.Monitor) model {
	logsVp := viewport.New(10, 10)
	clientsVp := viewport.New(10, 10)
	return model{
		monitor:     monitor,
		styles:      newStyles(),
		keys:        newKeyMap(),
		help:        newHelpModel(),
		view:        viewOverview,
		logsView:    logsVp,
		clientsView: clientsVp,
	}
}

func (m model) Init() tea.Cmd {
	return tickSnapshot(m.monitor)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewports()
		return m, nil

	case snapshotMsg:
		m.last = msg.snapshot
		m.updateViewports()
		return m, tickSnapshot(m.monitor)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Tab):
			m.view = (m.view + 1) % 3
			return m, nil
		case key.Matches(msg, m.keys.Reset):
			m.monitor.ResetSessionStats()
			return m, nil
		case key.Matches(msg, m.keys.Logs):
			m.view = viewLogs
			return m, nil
		case key.Matches(msg, m.keys.Clients):
			m.view = viewClients
			return m, nil
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			m.help.ShowAll = m.showHelp
			return m, nil
		}

		switch m.view {
		case viewLogs:
			var cmd tea.Cmd
			m.logsView, cmd = m.logsView.Update(msg)
			return m, cmd
		case viewClients:
			var cmd tea.Cmd
			m.clientsView, cmd = m.clientsView.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "initializing monitor..."
	}

	palette := panels.Palette{
		Title: m.styles.palette.title,
		Label: m.styles.palette.label,
		Value: m.styles.palette.value,
		Muted: m.styles.palette.muted,
		OK:    m.styles.palette.ok,
		Info:  m.styles.palette.info,
		Warn:  m.styles.palette.warn,
		Error: m.styles.palette.error,
	}

	header := panels.Header(m.last, m.width-2, palette)
	tabs := m.renderTabs()
	body := m.renderBody(palette)
	footer := m.help.View(m.keys)

	return m.styles.app.Render(
		lipgloss.JoinVertical(lipgloss.Left, header, tabs, body, footer),
	)
}

func (m model) renderTabs() string {
	items := []struct {
		label string
		mode  viewMode
	}{
		{"Overview", viewOverview},
		{"Logs", viewLogs},
		{"Clients", viewClients},
	}

	out := make([]string, 0, len(items))
	for _, item := range items {
		style := m.styles.tab
		if m.view == item.mode {
			style = m.styles.tabActive
		}
		out = append(out, style.Render(item.label))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, out...)
}

func (m model) renderBody(p panels.Palette) string {
	switch m.view {
	case viewLogs:
		return m.styles.panelFocused.Width(m.width - 2).Height(max(m.height-8, 8)).Render(
			m.styles.title.Render("Logs Live") + "\n" + m.logsView.View(),
		)
	case viewClients:
		return m.styles.panelFocused.Width(m.width - 2).Height(max(m.height-8, 8)).Render(
			m.styles.title.Render("Clients Connectes") + "\n" + m.clientsView.View(),
		)
	default:
		return m.renderOverview(p)
	}
}

func (m model) renderOverview(p panels.Palette) string {
	leftWidth := max((m.width-4)*60/100, 40)
	rightWidth := max((m.width-4)-leftWidth, 30)

	left := lipgloss.JoinVertical(
		lipgloss.Left,
		panels.WebSocket(m.last, leftWidth, false, p),
		panels.Upstream(m.last, leftWidth, false, p),
		panels.System(m.last, leftWidth, false, p),
	)

	right := lipgloss.JoinVertical(
		lipgloss.Left,
		panels.Alerts(m.last, rightWidth, false, p),
		panels.Clients(m.last.WS.Clients, rightWidth, false, p),
		panels.Logs(m.last.Logs, rightWidth, 10, false, p),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m *model) resizeViewports() {
	bodyHeight := max(m.height-10, 8)
	bodyWidth := max(m.width-6, 20)
	m.logsView.Width = bodyWidth
	m.logsView.Height = bodyHeight
	m.clientsView.Width = bodyWidth
	m.clientsView.Height = bodyHeight
}

func (m *model) updateViewports() {
	logLines := make([]string, 0, len(m.last.Logs))
	for _, entry := range m.last.Logs {
		logLines = append(logLines, fmt.Sprintf(
			"%s %-5s %-8s %s",
			entry.Time.Format("15:04:05"),
			entry.Severity,
			entry.Component,
			entry.Message,
		))
	}
	m.logsView.SetContent(strings.Join(logLines, "\n"))
	m.logsView.GotoBottom()

	clientLines := []string{"ID              REMOTE              STATE        LAST SEEN   CONNECTED"}
	for _, client := range m.last.WS.Clients {
		clientLines = append(clientLines, fmt.Sprintf(
			"%-14s %-18s %-12s %-10s %s",
			trim(client.ID, 14),
			trim(client.RemoteAddr, 18),
			client.State,
			shortAgo(client.LastSeenAt),
			shortAgo(client.ConnectedAt),
		))
	}
	m.clientsView.SetContent(strings.Join(clientLines, "\n"))
}

func tickSnapshot(monitor *monitoring.Monitor) tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return snapshotMsg{snapshot: monitor.Snapshot()}
	})
}

func shortAgo(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return time.Since(t).Truncate(time.Second).String()
}

func trim(v string, max int) string {
	if max <= 3 || len(v) <= max {
		return v
	}
	return v[:max-3] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
