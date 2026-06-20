package monitoring

import (
	"time"

	"veltryx/internal/logbuffer"
	"veltryx/internal/metrics"
)

type Snapshot struct {
	Service metrics.ServiceInfo
	WS      metrics.WebSocketStats
	X2      metrics.X2Stats
	System  metrics.SystemStats
	Alerts  []metrics.Alert
	Logs    []logbuffer.Entry
}

func (s Snapshot) Uptime() time.Duration {
	return s.Service.Now.Sub(s.Service.StartedAt)
}
