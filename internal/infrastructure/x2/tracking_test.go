package x2

import (
	"encoding/json"
	"math"
	"testing"
)

func TestParseTrackConfiguration(t *testing.T) {
	raw := map[string]any{
		"info": map[string]any{
			"version": 1,
			"name":    "Track A",
			"paths":   []any{},
			"lines":   []any{},
			"sectors": []any{},
		},
	}

	track := parseTrackConfiguration(raw)
	if track == nil {
		t.Fatalf("expected track configuration")
	}
	if track.Name != "Track A" || track.Version != 1 {
		t.Fatalf("unexpected track parsed: %+v", track)
	}
}

func TestHandleTrackingPayloadGPS(t *testing.T) {
	c := NewClient("host", "user", "pwd", "test")
	msg := []byte(`{
		"gps":[
			{"id":42,"lat":48.8566,"lon":2.3522,"speed":123.4,"timestamp":1700000,"vehicle":"#42"},
			{"id":99,"lat":91,"lon":0}
		]
	}`)

	if handled := c.handleTrackingPayload(msg); !handled {
		t.Fatalf("expected tracking payload to be handled")
	}

	s := c.Snapshot()
	if len(s.GPSLatest) != 1 {
		t.Fatalf("expected 1 valid GPS point, got %d", len(s.GPSLatest))
	}
	if s.GPSLatest[0].RacelinkID != 42 {
		t.Fatalf("unexpected GPS point: %+v", s.GPSLatest[0])
	}
}

func TestIngestGPSPointAliases(t *testing.T) {
	c := NewClient("host", "user", "pwd", "test")
	c.ingestGPSPoint(map[string]any{
		"racelinkid": 5.0,
		"latitude":   43.6,
		"longitude":  1.44,
		"vehicle":    "car5",
	})

	s := c.Snapshot()
	if len(s.GPSLatest) != 1 {
		t.Fatalf("expected aliased GPS point to be ingested")
	}
	if s.GPSLatest[0].RacelinkID != 5 || s.GPSLatest[0].Vehicle != "car5" {
		t.Fatalf("unexpected point from aliases: %+v", s.GPSLatest[0])
	}
}

func TestConvertersAndFinite(t *testing.T) {
	if got := toInt(json.Number("12")); got != 12 {
		t.Fatalf("toInt(json.Number) = %d, want 12", got)
	}
	if got := toFloat(json.Number("12.5")); got != 12.5 {
		t.Fatalf("toFloat(json.Number) = %v, want 12.5", got)
	}
	if got := toString(123); got != "" {
		t.Fatalf("toString(non string) = %q, want empty", got)
	}

	if !isFiniteLatLon(0, 0) {
		t.Fatalf("expected origin to be valid")
	}
	if isFiniteLatLon(math.NaN(), 0) {
		t.Fatalf("expected NaN lat to be invalid")
	}
	if isFiniteLatLon(45, 181) {
		t.Fatalf("expected lon out of range to be invalid")
	}
}
