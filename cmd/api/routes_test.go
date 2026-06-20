package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"veltryx/internal/domain/models"
	"veltryx/internal/infrastructure/desktopsession"
	"veltryx/internal/infrastructure/x2"
)

type mockX2Client struct {
	snapshot x2.Snapshot

	connectErr    error
	disconnectErr error
	globalFlagErr error
	sectorFlagErr error
	sendCanErr    error

	connectCalls    int
	disconnectCalls int
	globalFlagCalls int
	sectorFlagCalls int
	sendCanCalls    int
	lastSectorID    int
	lastSectorFlag  int
	lastGlobalFlag  int
	lastCanMessage  models.CanMessage
}

func (m *mockX2Client) Snapshot() x2.Snapshot { return m.snapshot }
func (m *mockX2Client) Connect() error {
	m.connectCalls++
	if m.connectErr != nil {
		return m.connectErr
	}
	m.snapshot.Connected = true
	if m.snapshot.State == "" {
		m.snapshot.State = "ready"
	}
	return nil
}
func (m *mockX2Client) Close() error {
	m.disconnectCalls++
	if m.disconnectErr != nil {
		return m.disconnectErr
	}
	m.snapshot.Connected = false
	m.snapshot.State = "disconnected"
	return nil
}
func (m *mockX2Client) IsConnected() bool { return m.snapshot.Connected }
func (m *mockX2Client) State() string     { return m.snapshot.State }
func (m *mockX2Client) SetGlobalFlag(flag int) error {
	m.globalFlagCalls++
	m.lastGlobalFlag = flag
	return m.globalFlagErr
}
func (m *mockX2Client) SetSectorFlag(sectorID int, flag int) error {
	m.sectorFlagCalls++
	m.lastSectorID = sectorID
	m.lastSectorFlag = flag
	return m.sectorFlagErr
}
func (m *mockX2Client) SendCan(message models.CanMessage) error {
	m.sendCanCalls++
	m.lastCanMessage = message
	return m.sendCanErr
}

type mockLiveClient struct {
	snapshot        models.LiveTimingSnapshot
	globalFlagCalls int
	lastGlobalFlag  int
	globalFlagErr   error
}

func (m *mockLiveClient) Snapshot() models.LiveTimingSnapshot { return m.snapshot }
func (m *mockLiveClient) SendGlobalFlag(flag int) error {
	m.globalFlagCalls++
	m.lastGlobalFlag = flag
	return m.globalFlagErr
}

type mockSessionStore struct {
	records map[string]desktopsession.Record
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{records: make(map[string]desktopsession.Record)}
}

func (m *mockSessionStore) Register(record desktopsession.Record) desktopsession.Record {
	if record.IssuedAt.IsZero() {
		record.IssuedAt = time.Now().UTC()
	}
	if record.ExpiresAt.IsZero() {
		record.ExpiresAt = time.Now().UTC().Add(12 * time.Hour)
	}
	if record.ServerSignature == "" {
		record.ServerSignature = "sig"
	}
	m.records[record.SessionID] = record
	return record
}

func (m *mockSessionStore) Validate(sessionID, deviceID, lastKnownUser string) (desktopsession.Record, bool, string) {
	rec, ok := m.records[sessionID]
	if !ok {
		return desktopsession.Record{}, false, "session_not_found"
	}
	return rec, true, ""
}

func (m *mockSessionStore) List() []desktopsession.Record {
	out := make([]desktopsession.Record, 0, len(m.records))
	for _, rec := range m.records {
		out = append(out, rec)
	}
	return out
}

func (m *mockSessionStore) Delete(sessionID string) {
	delete(m.records, sessionID)
}

func performJSON(t *testing.T, h http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestRoutes_X2StatusPermissionAndSuccess(t *testing.T) {
	mockX2 := &mockX2Client{snapshot: x2.Snapshot{Connected: true, State: "ready"}}
	mockLive := &mockLiveClient{}
	store := newMockSessionStore()
	mux := newAPIMux(mockX2, mockLive, store)

	unauth := performJSON(t, mux, http.MethodGet, "/api/x2/status", nil, nil)
	if unauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauth.Code)
	}

	ok := performJSON(t, mux, http.MethodGet, "/api/x2/status", nil, map[string]string{
		"X-Veltryx-Permissions": "websocket_access",
	})
	if ok.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", ok.Code)
	}
}

func TestRoutes_X2ConnectDisconnectAndFlags(t *testing.T) {
	mockX2 := &mockX2Client{snapshot: x2.Snapshot{Connected: true, State: "ready"}}
	live := &mockLiveClient{}
	mux := newAPIMux(mockX2, live, newMockSessionStore())

	connect := performJSON(t, mux, http.MethodPost, "/api/x2/connect", map[string]any{}, map[string]string{
		"X-Veltryx-Permissions": "x2_connection",
	})
	if connect.Code != http.StatusOK || mockX2.connectCalls != 1 {
		t.Fatalf("expected successful connect, code=%d calls=%d", connect.Code, mockX2.connectCalls)
	}

	global := performJSON(t, mux, http.MethodPost, "/api/x2/flags/global", map[string]any{"flag": 3}, map[string]string{
		"X-Veltryx-Permissions": "control_global_flags",
	})
	if global.Code != http.StatusOK || mockX2.lastGlobalFlag != 3 || live.globalFlagCalls != 1 || live.lastGlobalFlag != 3 {
		t.Fatalf(
			"expected global flag calls, code=%d x2Flag=%d risCalls=%d risFlag=%d",
			global.Code,
			mockX2.lastGlobalFlag,
			live.globalFlagCalls,
			live.lastGlobalFlag,
		)
	}

	sectorMissing := performJSON(t, mux, http.MethodPost, "/api/x2/flags/sector", map[string]any{"flag": 2}, map[string]string{
		"X-Veltryx-Permissions": "control_sector_flags",
	})
	if sectorMissing.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing sector, got %d", sectorMissing.Code)
	}

	sector := performJSON(t, mux, http.MethodPost, "/api/x2/flags/sector", map[string]any{"sectorId": 4, "flag": 2}, map[string]string{
		"X-Veltryx-Permissions": "control_sector_flags",
	})
	if sector.Code != http.StatusOK || mockX2.lastSectorID != 4 || mockX2.lastSectorFlag != 2 {
		t.Fatalf("expected sector flag call, code=%d sector=%d flag=%d", sector.Code, mockX2.lastSectorID, mockX2.lastSectorFlag)
	}

	can := performJSON(t, mux, http.MethodPost, "/api/x2/send-can", map[string]any{
		"id":    101,
		"canId": 514,
		"data":  []int{160, 2, 255, 64, 132, 81, 30, 61},
	}, map[string]string{
		"X-Veltryx-Permissions": "x2_connection",
	})
	if can.Code != http.StatusOK || mockX2.sendCanCalls != 1 {
		t.Fatalf("expected send-can call, code=%d calls=%d", can.Code, mockX2.sendCanCalls)
	}
	if mockX2.lastCanMessage.ID != 101 || mockX2.lastCanMessage.CanID != 514 || len(mockX2.lastCanMessage.Data) != 8 {
		t.Fatalf("unexpected can message: %+v", mockX2.lastCanMessage)
	}

	disconnect := performJSON(t, mux, http.MethodPost, "/api/x2/disconnect", map[string]any{}, map[string]string{
		"X-Veltryx-Permissions": "x2_connection",
	})
	if disconnect.Code != http.StatusOK || mockX2.disconnectCalls != 1 {
		t.Fatalf("expected successful disconnect, code=%d calls=%d", disconnect.Code, mockX2.disconnectCalls)
	}
}

func TestRoutes_X2Errors(t *testing.T) {
	mockX2 := &mockX2Client{
		connectErr:    errors.New("connect failed"),
		globalFlagErr: errors.New("flag failed"),
	}
	mux := newAPIMux(mockX2, &mockLiveClient{}, newMockSessionStore())

	connect := performJSON(t, mux, http.MethodPost, "/api/x2/connect", map[string]any{}, map[string]string{
		"X-Veltryx-Permissions": "x2_connection",
	})
	if connect.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 on connect error, got %d", connect.Code)
	}

	global := performJSON(t, mux, http.MethodPost, "/api/x2/flags/global", map[string]any{"flag": 1}, map[string]string{
		"X-Veltryx-Permissions": "control_global_flags",
	})
	if global.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 on flag error, got %d", global.Code)
	}
}

func TestRoutes_LiveTimingEndpoints(t *testing.T) {
	live := &mockLiveClient{
		snapshot: models.LiveTimingSnapshot{
			Stream: models.LiveTimingStreamSnapshot{
				Connected: true,
				Address:   "127.0.0.1:2058",
			},
			Participants: 2,
		},
	}
	mux := newAPIMux(&mockX2Client{}, live, newMockSessionStore())

	status := performJSON(t, mux, http.MethodGet, "/api/livetiming/status", nil, map[string]string{
		"X-Veltryx-Permissions": "view_live_timing",
	})
	if status.Code != http.StatusOK {
		t.Fatalf("expected 200 for livetiming status, got %d", status.Code)
	}

	snapshot := performJSON(t, mux, http.MethodGet, "/api/livetiming/snapshot", nil, map[string]string{
		"X-Veltryx-Permissions": "view_live_timing",
	})
	if snapshot.Code != http.StatusOK {
		t.Fatalf("expected 200 for livetiming snapshot, got %d", snapshot.Code)
	}
}

func TestRoutes_ClientSessionFlow(t *testing.T) {
	store := newMockSessionStore()
	mux := newAPIMux(&mockX2Client{}, &mockLiveClient{}, store)

	register := performJSON(t, mux, http.MethodPost, "/api/client-session/register", map[string]any{
		"session_id": "sess-1",
		"device_id":  "dev-1",
		"user": map[string]any{
			"id":          "u1",
			"role":        "viewer",
			"username":    "john",
			"permissions": []string{"view_live_timing"},
		},
	}, nil)
	if register.Code != http.StatusOK {
		t.Fatalf("expected 200 register, got %d", register.Code)
	}

	validate := performJSON(t, mux, http.MethodPost, "/api/client-session/validate", map[string]any{
		"session_id":      "sess-1",
		"device_id":       "dev-1",
		"last_known_user": "u1",
	}, nil)
	if validate.Code != http.StatusOK {
		t.Fatalf("expected 200 validate, got %d", validate.Code)
	}

	list := performJSON(t, mux, http.MethodGet, "/api/client-session/list", nil, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d", list.Code)
	}

	logout := performJSON(t, mux, http.MethodPost, "/api/client-session/logout", map[string]any{
		"session_id": "sess-1",
	}, nil)
	if logout.Code != http.StatusOK {
		t.Fatalf("expected 200 logout, got %d", logout.Code)
	}
}

func TestRoutes_StreamDeckLegacyEndpoints(t *testing.T) {
	mockX2 := &mockX2Client{
		snapshot: x2.Snapshot{
			Sectors: []models.SectorState{
				{ID: 2, Name: "S2"},
				{ID: 1, Name: "S1"},
			},
		},
	}
	mux := newAPIMux(mockX2, &mockLiveClient{}, newMockSessionStore())

	connect := performJSON(t, mux, http.MethodPost, "/connect", nil, nil)
	if connect.Code != http.StatusOK || mockX2.connectCalls != 1 {
		t.Fatalf("expected legacy connect to succeed, code=%d calls=%d", connect.Code, mockX2.connectCalls)
	}

	global := performJSON(t, mux, http.MethodPost, "/globalflag", map[string]any{
		"globalflag": map[string]any{"flag": 11},
	}, nil)
	if global.Code != http.StatusOK || mockX2.lastGlobalFlag != 11 {
		t.Fatalf("expected legacy globalflag to succeed, code=%d flag=%d", global.Code, mockX2.lastGlobalFlag)
	}

	sector := performJSON(t, mux, http.MethodPost, "/sector/flag", map[string]any{
		"id":   3,
		"flag": 2,
	}, nil)
	if sector.Code != http.StatusOK || mockX2.lastSectorID != 3 || mockX2.lastSectorFlag != 2 {
		t.Fatalf("expected legacy sector flag to succeed, code=%d sector=%d flag=%d", sector.Code, mockX2.lastSectorID, mockX2.lastSectorFlag)
	}

	sectors := performJSON(t, mux, http.MethodGet, "/streamdeck/sectors", nil, nil)
	if sectors.Code != http.StatusOK {
		t.Fatalf("expected sectors list to succeed, got %d", sectors.Code)
	}

	var payload []models.SectorState
	if err := json.Unmarshal(sectors.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode sectors payload: %v", err)
	}
	if len(payload) != 2 || payload[0].ID != 1 || payload[1].ID != 2 {
		t.Fatalf("expected sorted sectors, got %+v", payload)
	}
}

func TestRoutes_StreamDeckSectorsFallbackFromTrack(t *testing.T) {
	mockX2 := &mockX2Client{
		snapshot: x2.Snapshot{
			Track: &models.TrackConfiguration{
				Sectors: []models.TrackSector{
					{ID: 3, Name: "Sector 3"},
					{ID: 1, Name: "Sector 1"},
				},
			},
		},
	}
	mux := newAPIMux(mockX2, &mockLiveClient{}, newMockSessionStore())

	sectors := performJSON(t, mux, http.MethodGet, "/streamdeck/sectors", nil, nil)
	if sectors.Code != http.StatusOK {
		t.Fatalf("expected sectors list to succeed, got %d", sectors.Code)
	}

	var payload []models.SectorState
	if err := json.Unmarshal(sectors.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode sectors payload: %v", err)
	}
	if len(payload) != 2 || payload[0].ID != 3 || payload[1].ID != 1 {
		t.Fatalf("expected track order sectors from fallback, got %+v", payload)
	}
}
