package monitoring

import (
	"time"

	"veltryx/internal/logbuffer"
	"veltryx/internal/metrics"
)

type Monitor struct {
	serviceName string
	version     string
	startedAt   time.Time

	recorder *metrics.Recorder
	system   *metrics.SystemCollector
	logs     *logbuffer.Buffer
}

func New(serviceName, version string, recorder *metrics.Recorder, logs *logbuffer.Buffer) *Monitor {
	if recorder == nil {
		recorder = metrics.NewRecorder()
	}
	if logs == nil {
		logs = logbuffer.New(400)
	}
	return &Monitor{
		serviceName: serviceName,
		version:     version,
		startedAt:   time.Now(),
		recorder:    recorder,
		system:      metrics.NewSystemCollector(),
		logs:        logs,
	}
}

func (m *Monitor) Recorder() *metrics.Recorder {
	return m.recorder
}

func (m *Monitor) Logs() *logbuffer.Buffer {
	return m.logs
}

func (m *Monitor) Snapshot() Snapshot {
	now := time.Now()
	ws, x2, alerts := m.recorder.Snapshot(now)
	queueDepth, bufferedEvents := m.recorder.Counters()
	system := m.system.Snapshot(queueDepth, bufferedEvents)

	status := deriveServiceStatus(ws, x2, alerts)
	return Snapshot{
		Service: metrics.ServiceInfo{
			Name:        m.serviceName,
			Version:     m.version,
			StartedAt:   m.startedAt,
			Now:         now,
			Status:      status,
			StatusLabel: string(status),
		},
		WS:     ws,
		X2:     x2,
		System: system,
		Alerts: alerts,
		Logs:   m.logs.Snapshot(200),
	}
}

func (m *Monitor) ResetSessionStats() {
	m.recorder.ResetSessionStats()
	m.logs.Reset()
}

func deriveServiceStatus(ws metrics.WebSocketStats, x2 metrics.X2Stats, alerts []metrics.Alert) metrics.Health {
	if len(alerts) > 0 && alerts[len(alerts)-1].Severity == "error" {
		return metrics.HealthDegraded
	}
	if ws.Running && x2.Connected {
		return metrics.HealthConnected
	}
	if ws.Running || x2.Connected {
		return metrics.HealthDegraded
	}
	return metrics.HealthDisconnected
}
