package x2

import "testing"

func TestParseRaceLinks(t *testing.T) {
	raw := map[string]any{
		"list": []any{
			map[string]any{
				"id":       1.0,
				"name":     "RL-1",
				"flag":     2.0,
				"battery":  87.0,
				"rssi":     -70.0,
				"status":   "active",
				"bid":      3.0,
				"lastseen": 111.0,
				"car":      12.0,
			},
		},
	}

	out := parseRaceLinks(raw)
	if len(out) != 1 {
		t.Fatalf("expected 1 racelink, got %d", len(out))
	}
	if out[0].ID != 1 || out[0].Name != "RL-1" || out[0].CarNumber != 12 {
		t.Fatalf("unexpected parsed racelink: %+v", out[0])
	}
}

func TestParseRegistrations(t *testing.T) {
	raw := map[string]any{
		"list": []any{
			map[string]any{"racelink": 7.0, "number": 44.0, "firstname": "Jane", "lastname": "Doe"},
			map[string]any{"racelink": 0.0, "number": 11.0, "name": "skip"},
		},
	}

	out := parseRegistrations(raw)
	if len(out) != 1 {
		t.Fatalf("expected 1 registration, got %d", len(out))
	}
	if out[0].ID != 7 || out[0].CarNumber != 44 || out[0].Name != "Jane" {
		t.Fatalf("unexpected parsed registration: %+v", out[0])
	}
}

func TestParseSectorsAndAlerts(t *testing.T) {
	sectors := parseSectors(map[string]any{
		"list": []any{
			map[string]any{"id": 0.0, "name": "", "flag": 3.0},
		},
	})
	if len(sectors) != 1 {
		t.Fatalf("expected 1 sector, got %d", len(sectors))
	}
	if sectors[0].ID != 1 || sectors[0].Name != "S" || sectors[0].Flag != 3 {
		t.Fatalf("unexpected sector parsed: %+v", sectors[0])
	}

	alerts := parseAlerts(map[string]any{
		"list": []any{
			map[string]any{"code": "A1", "message": "Hello", "level": "warn"},
		},
	})
	if len(alerts) != 1 || alerts[0].Code != "A1" || alerts[0].Message != "Hello" {
		t.Fatalf("unexpected alerts parsed: %+v", alerts)
	}
}

func TestHandleOperationalPayload(t *testing.T) {
	c := NewClient("host", "user", "pwd", "test")
	msg := []byte(`{
		"registration":{"list":[{"racelink":21,"number":77,"name":"Car 77"}]},
		"globalflag":{"flag":4},
		"alerts":[{"code":"AL","message":"msg","level":"info"}]
	}`)

	if handled := c.handleOperationalPayload(msg); !handled {
		t.Fatalf("expected payload to be handled")
	}

	s := c.Snapshot()
	if s.GlobalFlag != 4 {
		t.Fatalf("expected global flag 4, got %d", s.GlobalFlag)
	}
	if len(s.RaceLinks) == 0 {
		t.Fatalf("expected racelink data in snapshot")
	}
	if len(s.Alerts) == 0 {
		t.Fatalf("expected alerts in snapshot")
	}
}
