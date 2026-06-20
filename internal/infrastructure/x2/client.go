package x2

import (
	"sync"
	"time"

	"veltryx/internal/domain/models"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn        *websocket.Conn
	state       string
	token       string
	host        string
	user        string
	password    string
	clientName  string
	done        chan struct{}
	writeMu     sync.Mutex
	mu          sync.RWMutex
	events      []Event
	totalMsgs   int
	lastMsg     int
	lastRaw     string
	lastAt      time.Time
	connectedAt time.Time
	welcome     *models.WelcomePayload
	auth        *models.AuthenticateResponse
	track       *models.TrackConfiguration
	gpsLatest   map[int]models.GPSPoint
	raceLinks   map[int]models.RaceLink
	baseLinks   []models.BaseLink
	sectors     map[int]models.SectorState
	globalFlag  int
	alerts      []models.AlertMessage
	subMu       sync.Mutex
	subscribers map[int]chan Snapshot
	nextSubID   int
}

type Event struct {
	Time    time.Time `json:"time"`
	Kind    string    `json:"kind"`
	Message string    `json:"message"`
	Raw     string    `json:"raw,omitempty"`
}

type Snapshot struct {
	Connected    bool                         `json:"connected"`
	State        string                       `json:"state"`
	TotalMsgs    int                          `json:"totalMsgs"`
	MessagesPerS float64                      `json:"messagesPerS"`
	LastMsg      int                          `json:"lastMsg"`
	LastRaw      string                       `json:"lastRaw"`
	LastAt       *time.Time                   `json:"lastAt,omitempty"`
	ConnectedAt  *time.Time                   `json:"connectedAt,omitempty"`
	Welcome      *models.WelcomePayload       `json:"welcome,omitempty"`
	Auth         *models.AuthenticateResponse `json:"auth,omitempty"`
	Events       []Event                      `json:"events"`
	Track        *models.TrackConfiguration   `json:"track,omitempty"`
	GPSLatest    []models.GPSPoint            `json:"gpsLatest,omitempty"`
	RaceLinks    []models.RaceLink            `json:"raceLinks,omitempty"`
	BaseLinks    []models.BaseLink            `json:"baseLinks,omitempty"`
	Sectors      []models.SectorState         `json:"sectors,omitempty"`
	GlobalFlag   int                          `json:"globalFlag,omitempty"`
	Alerts       []models.AlertMessage        `json:"alerts,omitempty"`
}

func NewClient(host, user, password, clientName string) *Client {
	return &Client{
		host:       host,
		user:       user,
		password:   password,
		clientName: clientName,
		state:      "disconnected",
		done:       make(chan struct{}),
		events:     make([]Event, 0, 64),
		gpsLatest:  make(map[int]models.GPSPoint),
		raceLinks:  make(map[int]models.RaceLink),
		sectors:    make(map[int]models.SectorState),
		subscribers: make(map[int]chan Snapshot),
	}
}

func (c *Client) SubscribeSnapshots() (int, <-chan Snapshot, func()) {
	ch := make(chan Snapshot, 2)

	c.subMu.Lock()
	c.nextSubID++
	id := c.nextSubID
	c.subscribers[id] = ch
	c.subMu.Unlock()

	// Send one immediate state to bootstrap the UI.
	ch <- c.Snapshot()

	unsubscribe := func() {
		c.subMu.Lock()
		if existing, ok := c.subscribers[id]; ok {
			delete(c.subscribers, id)
			close(existing)
		}
		c.subMu.Unlock()
	}

	return id, ch, unsubscribe
}

func (c *Client) PublishSnapshot() {
	snapshot := c.Snapshot()

	c.subMu.Lock()
	defer c.subMu.Unlock()

	for _, ch := range c.subscribers {
		select {
		case ch <- snapshot:
		default:
			// Keep the stream realtime by dropping stale frames.
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- snapshot:
			default:
			}
		}
	}
}

func (c *Client) appendEvent(kind, message, raw string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.events = append(c.events, Event{
		Time:    time.Now(),
		Kind:    kind,
		Message: message,
		Raw:     raw,
	})
	if len(c.events) > 50 {
		c.events = append([]Event(nil), c.events[len(c.events)-50:]...)
	}
}

func (c *Client) recordEnvelope(msg int, raw string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalMsgs++
	c.lastMsg = msg
	c.lastRaw = raw
	now := time.Now()
	c.lastAt = now
}

func (c *Client) setWelcome(welcome *models.WelcomePayload) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.welcome = welcome
}

func (c *Client) setAuth(auth *models.AuthenticateResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.auth = auth
}

func (c *Client) setState(state string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
}

func (c *Client) Snapshot() Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := Snapshot{
		Connected: c.conn != nil,
		State:     c.state,
		TotalMsgs: c.totalMsgs,
		LastMsg:   c.lastMsg,
		LastRaw:   c.lastRaw,
		Events:    append([]Event(nil), c.events...),
		Welcome:   c.welcome,
		Auth:      c.auth,
		Track:     c.track,
	}

	if len(c.gpsLatest) > 0 {
		snapshot.GPSLatest = make([]models.GPSPoint, 0, len(c.gpsLatest))
		for _, point := range c.gpsLatest {
			snapshot.GPSLatest = append(snapshot.GPSLatest, point)
		}
	}
	if len(c.raceLinks) > 0 {
		snapshot.RaceLinks = make([]models.RaceLink, 0, len(c.raceLinks))
		for _, item := range c.raceLinks {
			snapshot.RaceLinks = append(snapshot.RaceLinks, item)
		}
	}
	if len(c.baseLinks) > 0 {
		snapshot.BaseLinks = append([]models.BaseLink(nil), c.baseLinks...)
	}
	if len(c.sectors) > 0 {
		snapshot.Sectors = make([]models.SectorState, 0, len(c.sectors))
		for _, item := range c.sectors {
			snapshot.Sectors = append(snapshot.Sectors, item)
		}
	}
	if len(c.alerts) > 0 {
		snapshot.Alerts = append([]models.AlertMessage(nil), c.alerts...)
	}
	snapshot.GlobalFlag = c.globalFlag

	if !c.lastAt.IsZero() {
		t := c.lastAt
		snapshot.LastAt = &t
	}
	if !c.connectedAt.IsZero() {
		t := c.connectedAt
		snapshot.ConnectedAt = &t
		seconds := time.Since(c.connectedAt).Seconds()
		if seconds > 0 {
			snapshot.MessagesPerS = float64(c.totalMsgs) / seconds
		}
	}

	return snapshot
}

func (c *Client) setTrack(track *models.TrackConfiguration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.track = track
}

func (c *Client) updateGPSPoint(point models.GPSPoint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.gpsLatest == nil {
		c.gpsLatest = make(map[int]models.GPSPoint)
	}
	c.gpsLatest[point.RacelinkID] = point
}

func (c *Client) setRaceLinks(items []models.RaceLink) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.raceLinks == nil {
		c.raceLinks = make(map[int]models.RaceLink)
	}
	for _, item := range items {
		if item.ID == 0 {
			continue
		}
		current := c.raceLinks[item.ID]
		if item.Name != "" {
			current.Name = item.Name
		}
		if item.CarNumber != 0 {
			current.CarNumber = item.CarNumber
		}
		if item.Flag != 0 {
			current.Flag = item.Flag
		}
		if item.Battery != 0 {
			current.Battery = item.Battery
		}
		if item.RSSI != 0 {
			current.RSSI = item.RSSI
		}
		if item.Status != "" {
			current.Status = item.Status
		}
		if item.BaseLink != 0 {
			current.BaseLink = item.BaseLink
		}
		if item.LastSeen != 0 {
			current.LastSeen = item.LastSeen
		}
		current.ID = item.ID
		c.raceLinks[item.ID] = current
	}
}

func (c *Client) setBaseLinks(items []models.BaseLink) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseLinks = items
}

func (c *Client) setSectorStates(items []models.SectorState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sectors == nil {
		c.sectors = make(map[int]models.SectorState)
	}
	for _, item := range items {
		if item.ID == 0 {
			continue
		}
		current := c.sectors[item.ID]
		current.ID = item.ID
		if item.Name != "" {
			current.Name = item.Name
		}
		if item.Flag != 0 || current.Flag == 0 {
			current.Flag = item.Flag
		}
		c.sectors[item.ID] = current
	}
}

func (c *Client) setGlobalFlag(flag int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.globalFlag = flag
}

func (c *Client) setAlerts(items []models.AlertMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.alerts = items
}
