package metrics

import (
	"sort"
	"sync"
	"time"
)

type Recorder struct {
	mu sync.RWMutex

	wsRunning          bool
	wsListeningAddress string
	totalConnections   uint64
	totalMessagesIn    uint64
	totalMessagesOut   uint64
	totalBytesIn       uint64
	totalBytesOut      uint64

	lastRateAt      time.Time
	lastMessagesIn  uint64
	lastMessagesOut uint64
	lastBytesIn     uint64
	lastBytesOut    uint64

	msgInPerSecond    float64
	msgOutPerSecond   float64
	bytesInPerSecond  float64
	bytesOutPerSecond float64

	clients      map[string]ClientStats
	recentErrors []ErrorEvent
	alerts       []Alert

	x2Connected       bool
	x2Health          Health
	x2LastHeartbeatAt time.Time
	x2LastMessageAt   time.Time
	x2Reconnects      uint64
	x2LastSummary     string

	queueDepth     int
	bufferedEvents int
}

func NewRecorder() *Recorder {
	return &Recorder{
		lastRateAt: time.Now(),
		clients:    make(map[string]ClientStats),
	}
}

func (r *Recorder) SetWSListeningAddress(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.wsListeningAddress = addr
}

func (r *Recorder) SetWSRunning(running bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.wsRunning = running
}

func (r *Recorder) OnClientConnected(id, remoteAddr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	r.totalConnections++
	r.clients[id] = ClientStats{
		ID:          id,
		RemoteAddr:  remoteAddr,
		State:       "connected",
		ConnectedAt: now,
		LastSeenAt:  now,
	}
}

func (r *Recorder) OnClientDisconnected(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	client, ok := r.clients[id]
	if !ok {
		return
	}
	client.State = "disconnected"
	client.LastSeenAt = time.Now()
	r.clients[id] = client
}

func (r *Recorder) OnClientSeen(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	client, ok := r.clients[id]
	if !ok {
		return
	}
	client.LastSeenAt = time.Now()
	r.clients[id] = client
}

func (r *Recorder) OnWSMessageIn(id string, bytes int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.totalMessagesIn++
	r.totalBytesIn += uint64(max(bytes, 0))
	r.touchClientLocked(id, true)
}

func (r *Recorder) OnWSMessageOut(id string, bytes int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.totalMessagesOut++
	r.totalBytesOut += uint64(max(bytes, 0))
	r.touchClientLocked(id, false)
}

func (r *Recorder) RecordWSError(component, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	event := ErrorEvent{
		When:      time.Now(),
		Component: component,
		Message:   message,
	}
	r.recentErrors = append(r.recentErrors, event)
	if len(r.recentErrors) > 12 {
		r.recentErrors = append([]ErrorEvent(nil), r.recentErrors[len(r.recentErrors)-12:]...)
	}

	r.alerts = append(r.alerts, Alert{
		When:     event.When,
		Severity: "error",
		Title:    "WebSocket Error",
		Detail:   message,
	})
	if len(r.alerts) > 24 {
		r.alerts = append([]Alert(nil), r.alerts[len(r.alerts)-24:]...)
	}
}

func (r *Recorder) SetX2State(connected bool, health Health, summary string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.x2Connected = connected
	r.x2Health = health
	r.x2LastSummary = summary
}

func (r *Recorder) RecordX2Heartbeat() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.x2LastHeartbeatAt = time.Now()
}

func (r *Recorder) RecordX2Message(summary string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.x2LastMessageAt = time.Now()
	if summary != "" {
		r.x2LastSummary = summary
	}
}

func (r *Recorder) IncrementX2Reconnect() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.x2Reconnects++
	r.alerts = append(r.alerts, Alert{
		When:     time.Now(),
		Severity: "warn",
		Title:    "X2 Reconnect",
		Detail:   "Upstream reconnect detected",
	})
	if len(r.alerts) > 24 {
		r.alerts = append([]Alert(nil), r.alerts[len(r.alerts)-24:]...)
	}
}

func (r *Recorder) SetQueueDepth(depth int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queueDepth = max(depth, 0)
}

func (r *Recorder) SetBufferedEvents(count int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bufferedEvents = max(count, 0)
}

func (r *Recorder) ResetSessionStats() {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	r.totalConnections = 0
	r.totalMessagesIn = 0
	r.totalMessagesOut = 0
	r.totalBytesIn = 0
	r.totalBytesOut = 0
	r.lastMessagesIn = 0
	r.lastMessagesOut = 0
	r.lastBytesIn = 0
	r.lastBytesOut = 0
	r.lastRateAt = now
	r.msgInPerSecond = 0
	r.msgOutPerSecond = 0
	r.bytesInPerSecond = 0
	r.bytesOutPerSecond = 0
	r.recentErrors = nil
	r.alerts = nil
}

func (r *Recorder) Snapshot(now time.Time) (WebSocketStats, X2Stats, []Alert) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.computeRatesLocked(now)

	clients := make([]ClientStats, 0, len(r.clients))
	for _, client := range r.clients {
		clients = append(clients, client)
	}
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].ConnectedAt.Before(clients[j].ConnectedAt)
	})

	errors := append([]ErrorEvent(nil), r.recentErrors...)
	alerts := append([]Alert(nil), r.alerts...)

	ws := WebSocketStats{
		ListeningAddress:     r.wsListeningAddress,
		Running:              r.wsRunning,
		ConnectedClients:     countConnected(clients),
		TotalConnections:     r.totalConnections,
		MessagesInPerSecond:  r.msgInPerSecond,
		MessagesOutPerSecond: r.msgOutPerSecond,
		BytesInPerSecond:     r.bytesInPerSecond,
		BytesOutPerSecond:    r.bytesOutPerSecond,
		RecentErrors:         errors,
		Clients:              clients,
	}

	latestActivity := r.x2LastMessageAt
	if r.x2LastHeartbeatAt.After(latestActivity) {
		latestActivity = r.x2LastHeartbeatAt
	}

	x2 := X2Stats{
		Connected:          r.x2Connected,
		Health:             r.x2Health,
		LastHeartbeatAt:    r.x2LastHeartbeatAt,
		LastMessageAt:      r.x2LastMessageAt,
		Reconnects:         r.x2Reconnects,
		TimeSinceActivity:  time.Since(latestActivity),
		LastMessageSummary: r.x2LastSummary,
	}

	return ws, x2, alerts
}

func (r *Recorder) Counters() (queueDepth int, bufferedEvents int) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.queueDepth, r.bufferedEvents
}

func (r *Recorder) touchClientLocked(id string, inbound bool) {
	client, ok := r.clients[id]
	if !ok {
		client = ClientStats{
			ID:          id,
			RemoteAddr:  id,
			State:       "connected",
			ConnectedAt: time.Now(),
		}
	}
	client.LastSeenAt = time.Now()
	client.State = "connected"
	if inbound {
		client.InboundPerSecond = r.msgInPerSecond
	} else {
		client.OutboundPerSecond = r.msgOutPerSecond
	}
	r.clients[id] = client
}

func (r *Recorder) computeRatesLocked(now time.Time) {
	elapsed := now.Sub(r.lastRateAt).Seconds()
	if elapsed <= 0.2 {
		return
	}

	r.msgInPerSecond = float64(r.totalMessagesIn-r.lastMessagesIn) / elapsed
	r.msgOutPerSecond = float64(r.totalMessagesOut-r.lastMessagesOut) / elapsed
	r.bytesInPerSecond = float64(r.totalBytesIn-r.lastBytesIn) / elapsed
	r.bytesOutPerSecond = float64(r.totalBytesOut-r.lastBytesOut) / elapsed

	r.lastMessagesIn = r.totalMessagesIn
	r.lastMessagesOut = r.totalMessagesOut
	r.lastBytesIn = r.totalBytesIn
	r.lastBytesOut = r.totalBytesOut
	r.lastRateAt = now
}

func countConnected(clients []ClientStats) int {
	count := 0
	for _, client := range clients {
		if client.State == "connected" {
			count++
		}
	}
	return count
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
