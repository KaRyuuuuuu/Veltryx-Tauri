package metrics

import "time"

type Health string

const (
	HealthConnected    Health = "connected"
	HealthDegraded     Health = "degraded"
	HealthDisconnected Health = "disconnected"
)

type ClientStats struct {
	ID                string
	RemoteAddr        string
	State             string
	ConnectedAt       time.Time
	LastSeenAt        time.Time
	InboundPerSecond  float64
	OutboundPerSecond float64
}

type ErrorEvent struct {
	When      time.Time
	Component string
	Message   string
}

type Alert struct {
	When     time.Time
	Severity string
	Title    string
	Detail   string
}

type WebSocketStats struct {
	ListeningAddress     string
	Running              bool
	ConnectedClients     int
	TotalConnections     uint64
	MessagesInPerSecond  float64
	MessagesOutPerSecond float64
	BytesInPerSecond     float64
	BytesOutPerSecond    float64
	RecentErrors         []ErrorEvent
	Clients              []ClientStats
}

type X2Stats struct {
	Connected          bool
	Health             Health
	LastHeartbeatAt    time.Time
	LastMessageAt      time.Time
	Reconnects         uint64
	TimeSinceActivity  time.Duration
	LastMessageSummary string
}

type SystemStats struct {
	CPUPercent     float64
	MemoryUsedMB   uint64
	MemoryTotalMB  uint64
	MemoryUsagePct float64
	Goroutines     int
	OpenFileDesc   uint64
	QueueDepth     int
	BufferedEvents int
}

type ServiceInfo struct {
	Name        string
	Version     string
	StartedAt   time.Time
	Now         time.Time
	Status      Health
	StatusLabel string
}
