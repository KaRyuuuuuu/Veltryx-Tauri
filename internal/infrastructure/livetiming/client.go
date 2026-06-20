package livetiming

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"veltryx/internal/domain/models"
)

var messageMarkers = []string{
	"$ST;",
	"$PI;",
	"$CL;",
	"$I1;",
	"$I2;",
	"$M1;",
	"$M2;",
	"RT;",
}

type Client struct {
	address         string
	reconnectDelay  time.Duration
	mu              sync.RWMutex
	session         models.LiveTimingSessionInfo
	participants    map[string]models.LiveTimingParticipant
	classifications map[string]models.LiveTimingClassification
	messages        []models.LiveTimingRaceControlMessage
	stream          models.LiveTimingStreamSnapshot
}

func NewClient(address string) *Client {
	return &Client{
		address:         address,
		reconnectDelay:  2 * time.Second,
		participants:    make(map[string]models.LiveTimingParticipant),
		classifications: make(map[string]models.LiveTimingClassification),
		messages:        make([]models.LiveTimingRaceControlMessage, 0, 64),
		stream: models.LiveTimingStreamSnapshot{
			Address: address,
		},
	}
}

func (c *Client) Start() {
	go func() {
		for {
			if err := c.readLoop(); err != nil {
				c.setError(err)
			}
			time.Sleep(c.reconnectDelay)
		}
	}()
}

func (c *Client) Snapshot() models.LiveTimingSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	messages := append([]models.LiveTimingRaceControlMessage(nil), c.messages...)
	var lastMessage *models.LiveTimingRaceControlMessage
	if len(messages) > 0 {
		copyLast := messages[len(messages)-1]
		lastMessage = &copyLast
	}

	return models.LiveTimingSnapshot{
		Session:      c.session,
		Stream:       c.stream,
		Leaderboard:  buildLeaderboard(c.participants, c.classifications),
		LastMessage:  lastMessage,
		Messages:     messages,
		Participants: len(c.participants),
	}
}

func (c *Client) SendGlobalFlag(flag int) error {
	return c.sendJSON(map[string]any{
		"globalflag": map[string]any{
			"flag": flag,
		},
	})
}

func (c *Client) sendJSON(payload any) error {
	message, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal ris payload: %w", err)
	}
	message = append(message, '\n')

	conn, err := net.DialTimeout("tcp", c.address, 3*time.Second)
	if err != nil {
		return fmt.Errorf("dial ris %s: %w", c.address, err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return fmt.Errorf("set ris deadline: %w", err)
	}
	if _, err := conn.Write(message); err != nil {
		return fmt.Errorf("write ris payload: %w", err)
	}
	return nil
}

func (c *Client) readLoop() error {
	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return fmt.Errorf("dial %s: %w", c.address, err)
	}
	defer conn.Close()

	c.setConnected(true)

	buffer := make([]byte, 4096)
	pending := ""

	for {
		n, err := conn.Read(buffer)
		if n > 0 {
			chunk := normalizeChunk(string(buffer[:n]))
			c.markChunk(n)
			pending += chunk

			var messages []string
			messages, pending = extractMessages(pending)
			for _, message := range messages {
				c.markMessage(message)
				c.parseMessage(message)
			}
		}

		if err != nil {
			if err == io.EOF {
				c.setConnected(false)
				return nil
			}
			return fmt.Errorf("read stream: %w", err)
		}
	}
}

func (c *Client) setConnected(connected bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stream.Connected = connected
	if connected {
		c.stream.LastError = ""
	}
}

func (c *Client) setError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stream.Connected = false
	if err != nil {
		c.stream.LastError = err.Error()
	}
}

func (c *Client) markChunk(size int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stream.BytesReceived += int64(size)
	c.stream.LastChunkAt = time.Now()
}

func (c *Client) markMessage(message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stream.MessagesReceived++
	c.stream.LastMessage = message
	c.stream.RecentMessages = append(c.stream.RecentMessages, message)
	if len(c.stream.RecentMessages) > 120 {
		c.stream.RecentMessages = append([]string(nil), c.stream.RecentMessages[len(c.stream.RecentMessages)-120:]...)
	}
}

func (c *Client) parseMessage(msg string) {
	fields := splitFields(msg)
	if len(fields) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	switch fields[0] {
	case "$ST":
		if len(fields) >= 10 {
			c.session = models.LiveTimingSessionInfo{
				Name:        strings.TrimSpace(fields[1]),
				Series:      strings.TrimSpace(fields[2]),
				SessionType: strings.TrimSpace(fields[4]),
				Date:        strings.TrimSpace(fields[5]),
				Time:        strings.TrimSpace(fields[6]),
				Circuit:     strings.TrimSpace(fields[8]),
				DurationMs:  parseLiveTimingMs(fields[9]),
			}
		}
	case "$PI":
		if participant := parseParticipant(fields); participant.Number != "" {
			c.participants[participant.Number] = participant
		}
	case "$CL":
		if classif := parseClassification(fields); classif.CarNumber != "" {
			c.classifications[classif.CarNumber] = classif
		}
	case "$CH":
		c.parseChannel(fields)
	case "$M1", "$M2":
		message := parseRaceControlMessage(fields)
		if message.Tag != "" {
			c.messages = append(c.messages, message)
			if len(c.messages) > 40 {
				c.messages = append([]models.LiveTimingRaceControlMessage(nil), c.messages[len(c.messages)-40:]...)
			}
		}
	}
}

func splitFields(msg string) []string {
	rawFields := strings.Split(strings.TrimSpace(msg), ";")
	fields := make([]string, 0, len(rawFields))
	for _, field := range rawFields {
		fields = append(fields, strings.TrimSpace(field))
	}
	return fields
}

func normalizeChunk(chunk string) string {
	return strings.NewReplacer("\r", "", "\n", "", "\x00", "", "#", "#").Replace(chunk)
}

func extractMessages(stream string) ([]string, string) {
	stream = strings.TrimSpace(stream)
	if stream == "" {
		return nil, ""
	}

	if strings.Contains(stream, "#") {
		parts := strings.Split(stream, "#")
		messages := make([]string, 0, len(parts))
		for i := 0; i < len(parts)-1; i++ {
			message := strings.TrimSpace(parts[i])
			if message != "" {
				messages = append(messages, message)
			}
		}
		return messages, strings.TrimSpace(parts[len(parts)-1])
	}

	first := nextMarkerIndex(stream, 0)
	if first > 0 {
		stream = stream[first:]
	}
	if first == -1 {
		return nil, stream
	}

	indices := make([]int, 0, 8)
	for pos := 0; pos < len(stream); {
		idx := nextMarkerIndex(stream, pos)
		if idx == -1 {
			break
		}
		if len(indices) == 0 || indices[len(indices)-1] != idx {
			indices = append(indices, idx)
		}
		pos = idx + 1
	}

	if len(indices) <= 1 {
		return nil, stream
	}

	messages := make([]string, 0, len(indices)-1)
	for i := 0; i < len(indices)-1; i++ {
		message := strings.TrimSpace(stream[indices[i]:indices[i+1]])
		if message != "" {
			messages = append(messages, message)
		}
	}

	return messages, stream[indices[len(indices)-1]:]
}

func nextMarkerIndex(stream string, start int) int {
	best := -1
	for _, marker := range messageMarkers {
		idx := strings.Index(stream[start:], marker)
		if idx == -1 {
			continue
		}
		absolute := start + idx
		if best == -1 || absolute < best {
			best = absolute
		}
	}
	return best
}

func parseParticipant(fields []string) models.LiveTimingParticipant {
	participant := models.LiveTimingParticipant{}
	if len(fields) > 2 {
		participant.Number = normalizeCarNumber(fields[2])
	}
	if len(fields) > 4 {
		participant.Team = strings.TrimSpace(fields[4])
	}
	if len(fields) > 5 {
		participant.Car = strings.TrimSpace(fields[5])
	}
	if len(fields) > 23 {
		participant.Class = strings.TrimSpace(fields[23])
	}
	return participant
}

func parseClassification(fields []string) models.LiveTimingClassification {
	classif := models.LiveTimingClassification{}
	if len(fields) > 1 {
		classif.CarNumber = normalizeCarNumber(fields[1])
	}
	if len(fields) > 2 {
		classif.Laps = atoiSafe(fields[2])
	}
	if len(fields) > 9 {
		classif.Sector1Ms = sanitizeLapTime(parseLiveTimingMs(fields[9]))
	}
	if len(fields) > 10 {
		classif.Sector2Ms = sanitizeLapTime(parseLiveTimingMs(fields[10]))
	}
	if len(fields) > 11 {
		classif.Sector3Ms = sanitizeLapTime(parseLiveTimingMs(fields[11]))
	}
	if len(fields) > 8 {
		classif.LastLapMs = sanitizeLapTime(parseLiveTimingMs(fields[8]))
	}
	if len(fields) > 12 {
		classif.BestLapMs = sanitizeLapTime(parseLiveTimingMs(fields[12]))
	}
	if len(fields) > 19 {
		state := atoiSafe(fields[19])
		switch state {
		case 1:
			classif.PitState = 1
		case 3:
			classif.Status = 3
		}
	}
	return classif
}

func (c *Client) parseChannel(fields []string) {
	if len(fields) < 16 {
		return
	}

	carNumber := normalizeCarNumber(fields[1])
	if carNumber == "" {
		return
	}

	classif, ok := c.classifications[carNumber]
	if !ok {
		classif = parseClassification([]string{"$CL", carNumber})
	}

	classif.CarNumber = carNumber
	classif.Laps = atoiSafe(fields[2])
	if len(fields) > 3 {
		classif.PassageMs = int64(atoiSafe(fields[3]))
	}
	classif.BestLapMs = sanitizeLapTime(parseLiveTimingMs(fields[4]))

	if len(fields) > 6 {
		state := atoiSafe(fields[6])
		switch state {
		case 3:
			classif.Status = 3
		case 0:
			if classif.Status == 3 {
				classif.Status = 0
			}
		}
	}
	if len(fields) > 7 && atoiSafe(fields[7]) > 0 {
		classif.PitState = 1
	}
	if len(fields) > 8 {
		classif.Sector1Ms = sanitizeLapTime(parseLiveTimingMs(fields[8]))
	}
	if len(fields) > 9 {
		classif.Sector2Ms = sanitizeLapTime(parseLiveTimingMs(fields[9]))
	}
	if len(fields) > 15 {
		classif.GapMs = sanitizeLapTime(parseLiveTimingMs(fields[15]))
	}

	c.classifications[carNumber] = classif
}

func parseRaceControlMessage(fields []string) models.LiveTimingRaceControlMessage {
	message := models.LiveTimingRaceControlMessage{}
	if len(fields) > 0 {
		message.Tag = strings.TrimSpace(fields[0])
	}
	if len(fields) > 1 {
		message.Code = strings.TrimSpace(fields[1])
	}
	if len(fields) > 2 {
		message.Text = strings.TrimSpace(strings.Join(fields[2:], " "))
	}
	return message
}

func buildLeaderboard(participants map[string]models.LiveTimingParticipant, classifications map[string]models.LiveTimingClassification) []models.LiveTimingLeaderboardEntry {
	entries := make([]models.LiveTimingLeaderboardEntry, 0, len(classifications))
	for carNumber, classif := range classifications {
		participant := participants[carNumber]
		entries = append(entries, models.LiveTimingLeaderboardEntry{
			Position:  classif.Position,
			CarNumber: carNumber,
			Team:      participant.Team,
			Car:       participant.Car,
			Class:     participant.Class,
			Laps:      classif.Laps,
			PassageMs: classif.PassageMs,
			LastLapMs: classif.LastLapMs,
			BestLapMs: classif.BestLapMs,
			Sector1Ms: classif.Sector1Ms,
			Sector2Ms: classif.Sector2Ms,
			Sector3Ms: classif.Sector3Ms,
			GapMs:     classif.GapMs,
			Status:    classif.Status,
			PitState:  classif.PitState,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		left := entries[i].Position
		right := entries[j].Position
		if left == 0 && right > 0 {
			return false
		}
		if left > 0 && right == 0 {
			return true
		}
		if left > 0 && right > 0 {
			return left < right
		}
		if entries[i].Laps != entries[j].Laps {
			return entries[i].Laps > entries[j].Laps
		}
		if entries[i].GapMs != entries[j].GapMs {
			if entries[i].GapMs == 0 {
				return true
			}
			if entries[j].GapMs == 0 {
				return false
			}
			return entries[i].GapMs < entries[j].GapMs
		}
		return entries[i].CarNumber < entries[j].CarNumber
	})

	return entries
}
