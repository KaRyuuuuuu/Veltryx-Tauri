package panels

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"veltryx/internal/logbuffer"
	"veltryx/internal/metrics"
	"veltryx/internal/monitoring"
)

type Palette struct {
	Title, Label, Value, Muted string
	OK, Info, Warn, Error      string
}

func Header(snapshot monitoring.Snapshot, width int, p Palette) string {
	lines := []string{
		line("Service", snapshot.Service.Name, p),
		lineStyled("Status", withSeverity(p, severityFromHealth(snapshot.Service.Status)), p),
		line("Uptime", formatDuration(snapshot.Uptime()), p),
		line("Now", snapshot.Service.Now.Format("15:04:05"), p),
		line("Version", snapshot.Service.Version, p),
	}
	return box("Operations Dashboard", strings.Join(lines, "\n"), width, false, p)
}

func WebSocket(snapshot monitoring.Snapshot, width int, focused bool, p Palette) string {
	ws := snapshot.WS
	body := strings.Join([]string{
		line("Server", boolLabel(ws.Running), p),
		line("Listen", ws.ListeningAddress, p),
		line("Clients", fmt.Sprintf("%d", ws.ConnectedClients), p),
		line("Total Conn", fmt.Sprintf("%d", ws.TotalConnections), p),
		line("In Msg/s", fmt.Sprintf("%.1f", ws.MessagesInPerSecond), p),
		line("Out Msg/s", fmt.Sprintf("%.1f", ws.MessagesOutPerSecond), p),
		line("In BW", humanBytes(ws.BytesInPerSecond)+"/s", p),
		line("Out BW", humanBytes(ws.BytesOutPerSecond)+"/s", p),
		line("Recent Err", fmt.Sprintf("%d", len(ws.RecentErrors)), p),
	}, "\n")
	return box("WebSocket", body, width, focused, p)
}

func Upstream(snapshot monitoring.Snapshot, width int, focused bool, p Palette) string {
	x2 := snapshot.X2
	body := strings.Join([]string{
		lineStyled("State", withSeverity(p, severityFromHealth(x2.Health)), p),
		line("Connected", boolLabel(x2.Connected), p),
		line("Heartbeat", formatTimeAgo(x2.LastHeartbeatAt), p),
		line("Last Msg", formatTimeAgo(x2.LastMessageAt), p),
		line("Reconnects", fmt.Sprintf("%d", x2.Reconnects), p),
		line("Idle", formatDuration(x2.TimeSinceActivity), p),
		line("Summary", x2.LastMessageSummary, p),
	}, "\n")
	return box("X2 / Upstream", body, width, focused, p)
}

func System(snapshot monitoring.Snapshot, width int, focused bool, p Palette) string {
	sys := snapshot.System
	body := strings.Join([]string{
		line("CPU", fmt.Sprintf("%.1f%%", sys.CPUPercent), p),
		line("RAM", fmt.Sprintf("%d / %d MB", sys.MemoryUsedMB, sys.MemoryTotalMB), p),
		line("RAM %", fmt.Sprintf("%.1f%%", sys.MemoryUsagePct), p),
		line("Goroutines", fmt.Sprintf("%d", sys.Goroutines), p),
		line("FD", fmt.Sprintf("%d", sys.OpenFileDesc), p),
		line("Queues", fmt.Sprintf("%d", sys.QueueDepth), p),
		line("Buffered", fmt.Sprintf("%d", sys.BufferedEvents), p),
	}, "\n")
	return box("System", body, width, focused, p)
}

func Alerts(snapshot monitoring.Snapshot, width int, focused bool, p Palette) string {
	if len(snapshot.Alerts) == 0 {
		return box("Alerts", muted("No active alerts", p), width, focused, p)
	}
	lines := make([]string, 0, min(len(snapshot.Alerts), 6))
	for _, alert := range tailAlerts(snapshot.Alerts, 6) {
		lines = append(lines, fmt.Sprintf("%s %s", withSeverity(p, alert.Severity), trim(alert.Title+" - "+alert.Detail, width-10)))
	}
	return box("Alerts", strings.Join(lines, "\n"), width, focused, p)
}

func Logs(entries []logbuffer.Entry, width, height int, focused bool, p Palette) string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		severity := withSeverity(p, string(entry.Severity))
		lines = append(lines, fmt.Sprintf("%s %s %-8s %s", entry.Time.Format("15:04:05"), severity, entry.Component, entry.Message))
	}
	if len(lines) == 0 {
		lines = []string{muted("No logs yet", p)}
	}
	return box("Logs", strings.Join(lines, "\n"), width, focused, p)
}

func Clients(items []metrics.ClientStats, width int, focused bool, p Palette) string {
	if len(items) == 0 {
		return box("Clients", muted("No clients connected", p), width, focused, p)
	}
	lines := []string{"ID              STATE        LAST SEEN   UPTIME"}
	for _, item := range items {
		lines = append(lines, fmt.Sprintf(
			"%-14s %-12s %-10s %s",
			trim(item.ID, 14),
			item.State,
			formatTimeAgo(item.LastSeenAt),
			formatDuration(time.Since(item.ConnectedAt)),
		))
	}
	return box("Clients", strings.Join(lines, "\n"), width, focused, p)
}

func box(title, body string, width int, focused bool, p Palette) string {
	borderColor := p.Muted
	if focused {
		borderColor = p.OK
	}
	style := lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1)
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(p.Title)).Bold(true)
	return style.Render(titleStyle.Render(title) + "\n" + body)
}

func line(label, value string, palette Palette) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(palette.Label)).Render(label+": ") +
		lipgloss.NewStyle().Foreground(lipgloss.Color(palette.Value)).Bold(true).Render(value)
}

func lineStyled(label, value string, palette Palette) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(palette.Label)).Render(label+": ") + value
}

func withSeverity(p Palette, severity string) string {
	switch strings.ToLower(severity) {
	case "connected", "info":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(p.Info)).Bold(true).Render(strings.ToUpper(severity))
	case "warn", "degraded":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(p.Warn)).Bold(true).Render(strings.ToUpper(severity))
	case "error", "disconnected":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(p.Error)).Bold(true).Render(strings.ToUpper(severity))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(p.OK)).Bold(true).Render(strings.ToUpper(severity))
	}
}

func muted(value string, p Palette) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(p.Muted)).Render(value)
}

func severityFromHealth(h metrics.Health) string {
	switch h {
	case metrics.HealthConnected:
		return "connected"
	case metrics.HealthDegraded:
		return "warn"
	default:
		return "disconnected"
	}
}

func boolLabel(ok bool) string {
	if ok {
		return "yes"
	}
	return "no"
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return d.Truncate(time.Second).String()
	}
	if d < time.Hour {
		return d.Truncate(time.Second).String()
	}
	return d.Truncate(time.Minute).String()
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return formatDuration(time.Since(t)) + " ago"
}

func trim(v string, max int) string {
	if max <= 3 || len(v) <= max {
		return v
	}
	return v[:max-3] + "..."
}

func humanBytes(v float64) string {
	units := []string{"B", "KB", "MB", "GB"}
	idx := 0
	for v >= 1024 && idx < len(units)-1 {
		v /= 1024
		idx++
	}
	return fmt.Sprintf("%.1f %s", v, units[idx])
}

func tailAlerts(items []metrics.Alert, maxCount int) []metrics.Alert {
	if len(items) <= maxCount {
		return items
	}
	return items[len(items)-maxCount:]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
