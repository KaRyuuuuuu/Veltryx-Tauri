package monitoring

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"veltryx/internal/logbuffer"
	"veltryx/internal/metrics"
)

type MockFeed struct {
	recorder *metrics.Recorder
	logs     *logbuffer.Buffer
	rng      *rand.Rand
}

func NewMockFeed(recorder *metrics.Recorder, logs *logbuffer.Buffer) *MockFeed {
	return &MockFeed{
		recorder: recorder,
		logs:     logs,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (m *MockFeed) Run(ctx context.Context) {
	m.recorder.SetWSRunning(true)
	m.recorder.SetWSListeningAddress(":8080")
	m.recorder.SetX2State(true, metrics.HealthConnected, "stream snapshot received")

	clientTicker := time.NewTicker(3 * time.Second)
	eventTicker := time.NewTicker(300 * time.Millisecond)
	logTicker := time.NewTicker(1400 * time.Millisecond)
	defer clientTicker.Stop()
	defer eventTicker.Stop()
	defer logTicker.Stop()

	clientIDs := []string{"race-control-1", "viewer-bridge", "streamdeck-ops"}
	for _, id := range clientIDs {
		m.recorder.OnClientConnected(id, id+".local")
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-clientTicker.C:
			m.toggleRandomClient(clientIDs)
			m.recorder.IncrementX2Reconnect()
		case <-eventTicker.C:
			client := clientIDs[m.rng.Intn(len(clientIDs))]
			inBytes := 180 + m.rng.Intn(900)
			outBytes := 80 + m.rng.Intn(400)
			m.recorder.OnWSMessageIn(client, inBytes)
			m.recorder.OnWSMessageOut(client, outBytes)
			m.recorder.RecordX2Heartbeat()
			m.recorder.RecordX2Message(fmt.Sprintf("sector update #%d", m.rng.Intn(999)))
			m.recorder.SetQueueDepth(m.rng.Intn(8))
			m.recorder.SetBufferedEvents(m.rng.Intn(24))
		case <-logTicker.C:
			m.emitMockLog()
		}
	}
}

func (m *MockFeed) toggleRandomClient(ids []string) {
	id := ids[m.rng.Intn(len(ids))]
	if m.rng.Intn(10) > 5 {
		m.recorder.OnClientConnected(id, id+".local")
		m.logs.Addf(logbuffer.SeverityInfo, "ws", "client %s connected", id)
		return
	}
	m.recorder.OnClientDisconnected(id)
	m.recorder.RecordWSError("ws", fmt.Sprintf("client %s dropped unexpectedly", id))
	m.logs.Addf(logbuffer.SeverityWarn, "ws", "client %s disconnected", id)
}

func (m *MockFeed) emitMockLog() {
	switch m.rng.Intn(10) {
	case 0:
		m.logs.Add(logbuffer.SeverityError, "x2", "upstream latency spike detected")
	case 1, 2:
		m.logs.Add(logbuffer.SeverityWarn, "bridge", "backpressure rising on outbound queue")
	default:
		m.logs.Add(logbuffer.SeverityInfo, "ws", "snapshot published to subscribers")
	}
}
