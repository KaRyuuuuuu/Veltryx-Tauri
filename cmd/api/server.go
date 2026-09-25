package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"sync"
	"time"

	"veltryx/internal/domain/models"
	"veltryx/internal/infrastructure/desktopsession"
	"veltryx/internal/infrastructure/x2"

	"github.com/gorilla/websocket"
)

type x2API interface {
	Snapshot() x2.Snapshot
	Connect() error
	Close() error
	IsConnected() bool
	State() string
	SetGlobalFlag(flag int) error
	SetSectorFlag(sectorID int, flag int) error
	SendCan(message models.CanMessage) error
}

type liveTimingAPI interface {
	Snapshot() models.LiveTimingSnapshot
	SendGlobalFlag(flag int) error
}

type sessionStoreAPI interface {
	Register(record desktopsession.Record) desktopsession.Record
	Validate(sessionID, deviceID, lastKnownUser string) (desktopsession.Record, bool, string)
	List() []desktopsession.Record
	Delete(sessionID string)
}

const (
	sectorGreenFlag     = 5
	sectorClearFlag     = 0
	sectorAutoClearWait = 15 * time.Second
)

type sectorAutoClearScheduler struct {
	client   x2API
	delay    time.Duration
	clear    func(sectorID int) error
	mu       sync.Mutex
	versions map[int]uint64
}

func newSectorAutoClearScheduler(client x2API, delay time.Duration, clear func(sectorID int) error) *sectorAutoClearScheduler {
	return &sectorAutoClearScheduler{
		client:   client,
		delay:    delay,
		clear:    clear,
		versions: make(map[int]uint64),
	}
}

func (s *sectorAutoClearScheduler) Record(sectorID, flag int) {
	s.mu.Lock()
	s.versions[sectorID]++
	version := s.versions[sectorID]
	s.mu.Unlock()

	if flag != sectorGreenFlag {
		return
	}

	time.AfterFunc(s.delay, func() {
		s.mu.Lock()
		if s.versions[sectorID] != version {
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()

		snapshot := s.client.Snapshot()
		if previousSectorIsYellow(snapshot, sectorID) {
			return
		}

		if err := s.clear(sectorID); err == nil {
			s.mu.Lock()
			if s.versions[sectorID] == version {
				s.versions[sectorID]++
			}
			s.mu.Unlock()
		}
	})
}

func previousSectorIsYellow(snapshot x2.Snapshot, sectorID int) bool {
	orderedIDs := make([]int, 0)
	if snapshot.Track != nil {
		for _, sector := range snapshot.Track.Sectors {
			if sector.ID > 0 {
				orderedIDs = append(orderedIDs, sector.ID)
			}
		}
	}
	if len(orderedIDs) == 0 {
		for _, sector := range snapshot.Sectors {
			if sector.ID > 0 {
				orderedIDs = append(orderedIDs, sector.ID)
			}
		}
		sort.Ints(orderedIDs)
	}
	if len(orderedIDs) < 2 {
		return false
	}

	previousID := 0
	for index, id := range orderedIDs {
		if id == sectorID {
			previousID = orderedIDs[(index+len(orderedIDs)-1)%len(orderedIDs)]
			break
		}
	}
	if previousID == 0 {
		return false
	}
	for _, sector := range snapshot.Sectors {
		if sector.ID == previousID {
			return isYellowSectorFlag(sector.Flag)
		}
	}
	return false
}

func isYellowSectorFlag(flag int) bool {
	switch flag {
	case 2, 3, 32, 33, 34:
		return true
	default:
		return false
	}
}

func newAPIMux(client x2API, liveClient liveTimingAPI, sessionStore sessionStoreAPI) *http.ServeMux {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			return isAllowedOrigin(origin)
		},
	}
	bridgeUpgrader := newBridgeUpgrader()
	bridgeState := &bridgeStateStore{}

	type x2SnapshotStreamer interface {
		SubscribeSnapshots() (int, <-chan x2.Snapshot, func())
	}

	ensureX2Connected := func() error {
		if client.IsConnected() {
			deadline := time.Now().Add(3 * time.Second)
			for {
				state := client.State()
				if state == "ready" {
					return nil
				}
				if !client.IsConnected() {
					break
				}
				if time.Now().After(deadline) {
					return fmt.Errorf("x2 not ready (state=%s)", state)
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
		if err := client.Connect(); err != nil {
			return err
		}
		deadline := time.Now().Add(4 * time.Second)
		for {
			state := client.State()
			if state == "ready" {
				return nil
			}
			if !client.IsConnected() {
				return fmt.Errorf("x2 disconnected during startup")
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("x2 startup timeout (state=%s)", state)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}

	withX2Retry := func(fn func() error) error {
		if err := ensureX2Connected(); err != nil {
			return err
		}
		if err := fn(); err == nil {
			return nil
		}

		// Connection may be stale; force a reconnect path and wait for ready again.
		_ = client.Close()
		if err := ensureX2Connected(); err != nil {
			return err
		}
		return fn()
	}

	setSectorFlag := func(sectorID, flag int) error {
		return withX2Retry(func() error {
			return client.SetSectorFlag(sectorID, flag)
		})
	}
	autoClearSectors := newSectorAutoClearScheduler(client, sectorAutoClearWait, func(sectorID int) error {
		return setSectorFlag(sectorID, sectorClearFlag)
	})
	applySectorFlag := func(sectorID, flag int) error {
		if err := setSectorFlag(sectorID, flag); err != nil {
			return err
		}
		autoClearSectors.Record(sectorID, flag)
		return nil
	}

	withPermission := func(permission string, next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var granted []string
			granted = permissionsFromHeader(r.Header.Get("X-Veltryx-Permissions"))

			if len(granted) == 0 {
				sessionID := r.Header.Get("X-Veltryx-Session-ID")
				if sessionID != "" {
					records := sessionStore.List()
					for _, rec := range records {
						if rec.SessionID == sessionID {
							granted = rec.User.Permissions
							break
						}
					}
				}
			}

			if len(granted) == 0 {
				writeJSON(w, http.StatusUnauthorized, map[string]any{
					"error": "missing_desktop_permissions_or_session",
				})
				return
			}

			if !slices.Contains(granted, permission) {
				writeJSON(w, http.StatusForbidden, map[string]any{
					"error":      "permission_denied",
					"permission": permission,
				})
				return
			}

			next(w, r)
		}
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/x2/status", withPermission("websocket_access", func(w http.ResponseWriter, r *http.Request) {
		snapshot := client.Snapshot()
		writeJSON(w, http.StatusOK, map[string]any{
			"connected": snapshot.Connected,
			"state":     snapshot.State,
		})
	}))

	mux.HandleFunc("/api/x2/telemetry", withPermission("websocket_access", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.Snapshot())
	}))

	mux.HandleFunc("/api/x2/stream", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		streamer, ok := client.(x2SnapshotStreamer)
		if !ok {
			writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "stream_not_supported"})
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		_, ch, unsubscribe := streamer.SubscribeSnapshots()
		defer unsubscribe()

		pingTicker := time.NewTicker(15 * time.Second)
		defer pingTicker.Stop()

		for {
			select {
			case snapshot, ok := <-ch:
				if !ok {
					return
				}
				if err := conn.WriteJSON(snapshot); err != nil {
					return
				}
			case <-pingTicker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(2*time.Second)); err != nil {
					return
				}
			}
		}
	})

	mux.HandleFunc("/ws/bridge", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		conn, err := bridgeUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var env bridgeEncryptedEnvelope
			if err := json.Unmarshal(raw, &env); err != nil {
				_ = conn.WriteJSON(map[string]any{
					"ok":    false,
					"error": "invalid_envelope",
				})
				continue
			}

			payload, err := decryptBridgePayload(env)
			if err != nil {
				fmt.Printf("[bridge] decrypt error agent=%s err=%v\n", env.AgentID, err)
				_ = conn.WriteJSON(map[string]any{
					"ok":    false,
					"error": "decrypt_failed",
				})
				continue
			}

			bridgeState.Save(env.AgentID, payload)
			fmt.Printf(
				"[bridge] sync agent=%s global=%d sectors=%d at=%s\n",
				env.AgentID,
				payload.GlobalFlag.Code,
				len(payload.Sectors),
				payload.SentAt,
			)

			_ = conn.WriteJSON(map[string]any{
				"ok":         true,
				"receivedAt": time.Now().UTC().Format(time.RFC3339),
				"schema":     payload.Schema,
				"agentId":    env.AgentID,
			})
		}
	})

	mux.HandleFunc("/api/bridge/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, bridgeState.Snapshot())
	})

	mux.HandleFunc("/api/livetiming/status", withPermission("view_live_timing", func(w http.ResponseWriter, r *http.Request) {
		snapshot := liveClient.Snapshot()
		writeJSON(w, http.StatusOK, map[string]any{
			"connected": snapshot.Stream.Connected,
			"address":   snapshot.Stream.Address,
			"error":     snapshot.Stream.LastError,
		})
	}))

	mux.HandleFunc("/api/livetiming/snapshot", withPermission("view_live_timing", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, liveClient.Snapshot())
	}))

	mux.HandleFunc("/api/client-session/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			SessionID string              `json:"session_id"`
			DeviceID  string              `json:"device_id"`
			User      desktopsession.User `json:"user"`
			ExpiresAt time.Time           `json:"expires_at"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		if req.SessionID == "" || req.User.ID == "" || req.User.Role == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing_fields"})
			return
		}

		record := sessionStore.Register(desktopsession.Record{
			SessionID: req.SessionID,
			DeviceID:  req.DeviceID,
			User:      req.User,
			ExpiresAt: req.ExpiresAt,
		})
		writeJSON(w, http.StatusOK, map[string]any{
			"valid":            true,
			"session_id":       record.SessionID,
			"user":             record.User,
			"issued_at":        record.IssuedAt,
			"expires_at":       record.ExpiresAt,
			"server_signature": record.ServerSignature,
		})
	})

	mux.HandleFunc("/api/client-session/validate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			SessionID     string `json:"session_id"`
			DeviceID      string `json:"device_id"`
			LastKnownUser string `json:"last_known_user"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}

		record, ok, reason := sessionStore.Validate(req.SessionID, req.DeviceID, req.LastKnownUser)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"valid":  false,
				"reason": reason,
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"valid":            true,
			"session_id":       record.SessionID,
			"user":             record.User,
			"issued_at":        record.IssuedAt,
			"expires_at":       record.ExpiresAt,
			"server_signature": record.ServerSignature,
		})
	})

	mux.HandleFunc("/api/client-session/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		records := sessionStore.List()
		sessions := make([]map[string]any, 0, len(records))
		for _, record := range records {
			sessions = append(sessions, map[string]any{
				"sessionKey":  record.SessionID,
				"userId":      record.User.ID,
				"username":    record.User.Username,
				"name":        record.User.Name,
				"role":        record.User.Role,
				"tenantId":    record.User.TenantID,
				"clientLabel": record.DeviceID,
				"lastSeenAt":  record.ExpiresAt,
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"sessions": sessions,
			"total":    len(sessions),
		})
	})

	mux.HandleFunc("/api/client-session/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			SessionID string `json:"session_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		if req.SessionID != "" {
			sessionStore.Delete(req.SessionID)
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/x2/connect", withPermission("x2_connection", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := client.Connect(); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":        true,
			"connected": client.IsConnected(),
			"state":     client.State(),
		})
	}))

	mux.HandleFunc("/api/x2/disconnect", withPermission("x2_connection", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := client.Close(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":        true,
			"connected": client.IsConnected(),
			"state":     client.State(),
		})
	}))

	mux.HandleFunc("/api/x2/flags/global", withPermission("control_global_flags", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Flag int `json:"flag"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		if err := client.SetGlobalFlag(req.Flag); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		if err := liveClient.SendGlobalFlag(req.Flag); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"ok":     false,
				"error":  err.Error(),
				"target": "ris",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}))

	mux.HandleFunc("/api/x2/flags/sector", withPermission("control_sector_flags", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			SectorID int `json:"sectorId"`
			Flag     int `json:"flag"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		if req.SectorID == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing_sector"})
			return
		}
		if err := applySectorFlag(req.SectorID, req.Flag); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}))

	mux.HandleFunc("/api/x2/send-can", withPermission("x2_connection", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ID    int   `json:"id"`
			CanID int   `json:"canId"`
			Data  []int `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		if req.ID <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing_id"})
			return
		}
		if req.CanID <= 0 {
			req.CanID = 514
		}
		if len(req.Data) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing_data"})
			return
		}
		for _, value := range req.Data {
			if value < 0 || value > 255 {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_data_byte"})
				return
			}
		}
		if err := withX2Retry(func() error {
			return client.SendCan(models.CanMessage{
				ID:    req.ID,
				CanID: req.CanID,
				Data:  req.Data,
			})
		}); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":    true,
			"id":    req.ID,
			"canId": req.CanID,
			"data":  req.Data,
		})
	}))

	// Legacy Stream Deck compatibility routes (no desktop permission headers).
	mux.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		fmt.Printf("[streamdeck] /connect method=%s remote=%s\n", r.Method, r.RemoteAddr)
		if err := client.Connect(); err != nil {
			fmt.Printf("[streamdeck] /connect error=%v\n", err)
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}
		fmt.Printf("[streamdeck] /connect ok state=%s\n", client.State())
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	mux.HandleFunc("/globalflag", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Flag       *int `json:"flag"`
			GlobalFlag *struct {
				Flag *int `json:"flag"`
			} `json:"globalflag"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		flag := req.Flag
		if flag == nil && req.GlobalFlag != nil {
			flag = req.GlobalFlag.Flag
		}
		if flag == nil {
			fmt.Printf("[streamdeck] /globalflag invalid payload missing flag\n")
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing_flag"})
			return
		}
		fmt.Printf("[streamdeck] /globalflag flag=%d remote=%s\n", *flag, r.RemoteAddr)
		if err := withX2Retry(func() error { return client.SetGlobalFlag(*flag) }); err != nil {
			fmt.Printf("[streamdeck] /globalflag error=%v state=%s\n", err, client.State())
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"ok":    false,
				"error": err.Error(),
				"state": client.State(),
			})
			return
		}
		fmt.Printf("[streamdeck] /globalflag ok flag=%d state=%s\n", *flag, client.State())
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	mux.HandleFunc("/sector/flag", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ID       *int `json:"id"`
			SectorID *int `json:"sectorId"`
			Sector   *int `json:"sector"`
			Flag     *int `json:"flag"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}
		sectorID := req.ID
		if sectorID == nil {
			sectorID = req.SectorID
		}
		if sectorID == nil {
			sectorID = req.Sector
		}
		if sectorID == nil || *sectorID <= 0 {
			fmt.Printf("[streamdeck] /sector/flag invalid payload missing sector\n")
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing_sector"})
			return
		}
		if req.Flag == nil {
			fmt.Printf("[streamdeck] /sector/flag invalid payload missing flag sector=%d\n", *sectorID)
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing_flag"})
			return
		}
		fmt.Printf("[streamdeck] /sector/flag sector=%d flag=%d remote=%s\n", *sectorID, *req.Flag, r.RemoteAddr)
		if err := applySectorFlag(*sectorID, *req.Flag); err != nil {
			fmt.Printf("[streamdeck] /sector/flag error=%v state=%s sector=%d flag=%d\n", err, client.State(), *sectorID, *req.Flag)
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"ok":       false,
				"error":    err.Error(),
				"state":    client.State(),
				"sectorId": *sectorID,
				"flag":     *req.Flag,
			})
			return
		}
		fmt.Printf("[streamdeck] /sector/flag ok sector=%d flag=%d state=%s\n", *sectorID, *req.Flag, client.State())
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	mux.HandleFunc("/streamdeck/sectors", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		snapshot := client.Snapshot()
		sectorsByID := make(map[int]models.SectorState, len(snapshot.Sectors))
		for _, sector := range snapshot.Sectors {
			if sector.ID <= 0 {
				continue
			}
			sectorsByID[sector.ID] = sector
		}

		// Prefer track sector order for Stream Deck mapping (S1/S2/... visual order),
		// while keeping runtime flag values from live sector snapshots when available.
		sectors := make([]models.SectorState, 0, len(snapshot.Sectors))
		if snapshot.Track != nil && len(snapshot.Track.Sectors) > 0 {
			for _, sector := range snapshot.Track.Sectors {
				if sector.ID <= 0 {
					continue
				}
				name := sector.Name
				if name == "" {
					name = fmt.Sprintf("S %d", sector.ID)
				}
				flag := 0
				if live, ok := sectorsByID[sector.ID]; ok {
					if live.Name != "" {
						name = live.Name
					}
					flag = live.Flag
					delete(sectorsByID, sector.ID)
				}
				sectors = append(sectors, models.SectorState{
					ID:   sector.ID,
					Name: name,
					Flag: flag,
				})
			}
		}

		// Add remaining live sectors not present in track config.
		if len(sectorsByID) > 0 {
			extra := make([]models.SectorState, 0, len(sectorsByID))
			for _, sector := range sectorsByID {
				name := sector.Name
				if name == "" {
					name = fmt.Sprintf("S %d", sector.ID)
				}
				extra = append(extra, models.SectorState{
					ID:   sector.ID,
					Name: name,
					Flag: sector.Flag,
				})
			}
			sort.Slice(extra, func(i, j int) bool {
				return extra[i].ID < extra[j].ID
			})
			sectors = append(sectors, extra...)
		}

		if len(sectors) == 0 {
			sectors = make([]models.SectorState, 0, 20)
			for i := 1; i <= 20; i++ {
				sectors = append(sectors, models.SectorState{
					ID:   i,
					Name: fmt.Sprintf("S %d", i),
				})
			}
		}
		writeJSON(w, http.StatusOK, sectors)
	})

	return mux
}
