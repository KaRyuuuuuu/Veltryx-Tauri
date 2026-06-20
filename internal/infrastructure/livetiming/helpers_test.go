package livetiming

import "testing"

func TestNormalizeCarNumber(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"07", "7"},
		{"000", ""},
		{" 0012 ", "12"},
		{"A12", "A12"},
		{"", ""},
	}

	for _, tc := range cases {
		if got := normalizeCarNumber(tc.in); got != tc.want {
			t.Fatalf("normalizeCarNumber(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTimeParsing(t *testing.T) {
	if got := timeToMs("75.5"); got != 75500 {
		t.Fatalf("timeToMs seconds = %d, want 75500", got)
	}
	if got := timeToMs("1:02.500"); got != 62500 {
		t.Fatalf("timeToMs mm:ss = %d, want 62500", got)
	}
	if got := timeToMs("1:01:01.000"); got != 3661000 {
		t.Fatalf("timeToMs hh:mm:ss = %d, want 3661000", got)
	}
	if got := parseLiveTimingMs(" 1234 "); got != 1234 {
		t.Fatalf("parseLiveTimingMs int = %d, want 1234", got)
	}
	if got := parseLiveTimingMs("-"); got != 0 {
		t.Fatalf("parseLiveTimingMs dash = %d, want 0", got)
	}
}

func TestSanitizeLapTime(t *testing.T) {
	if got := sanitizeLapTime(0); got != 0 {
		t.Fatalf("sanitize 0 = %d, want 0", got)
	}
	if got := sanitizeLapTime(3599999); got != 0 {
		t.Fatalf("sanitize sentinel 3599999 = %d, want 0", got)
	}
	if got := sanitizeLapTime(215999999); got != 0 {
		t.Fatalf("sanitize sentinel 215999999 = %d, want 0", got)
	}
	if got := sanitizeLapTime(12*60*60*1000 + 1); got != 0 {
		t.Fatalf("sanitize >12h = %d, want 0", got)
	}
	if got := sanitizeLapTime(91234); got != 91234 {
		t.Fatalf("sanitize valid = %d, want 91234", got)
	}
}
