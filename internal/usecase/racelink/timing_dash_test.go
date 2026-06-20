package racelink

import (
	"errors"
	"testing"
)

func TestMapToX2_MatchPriority(t *testing.T) {
	t.Run("transponder_has_priority", func(t *testing.T) {
		mapper := &TimingToX2Mapper{
			LiveParticipants: []LiveTimingParticipant{
				{
					ID:             "live-1",
					Transponder:    "tx-100",
					RegistrationID: "reg-a",
					CarNumber:      "07",
					DriverName:     "John Doe",
					TeamName:       "VDS Racing",
				},
			},
			X2Participants: []X2Participant{
				{
					ID:             "x2-transponder",
					Transponder:    "TX-100",
					RegistrationID: "reg-other",
					CarNumber:      "99",
				},
				{
					ID:             "x2-registration",
					RegistrationID: "REG-A",
					CarNumber:      "7",
				},
			},
		}

		MapToX2(mapper)

		result, ok := mapper.Results["live-1"]
		if !ok {
			t.Fatalf("expected result for live-1")
		}
		if result.X2 == nil {
			t.Fatalf("expected an X2 match")
		}
		if result.X2.ID != "x2-transponder" {
			t.Fatalf("expected transponder match, got %q", result.X2.ID)
		}
		if result.Strategy != MatchByTransponder {
			t.Fatalf("expected strategy %q, got %q", MatchByTransponder, result.Strategy)
		}
	})

	t.Run("registration_fallback", func(t *testing.T) {
		mapper := &TimingToX2Mapper{
			LiveParticipants: []LiveTimingParticipant{
				{
					ID:             "live-2",
					RegistrationID: "REG-42",
					CarNumber:      "9",
				},
			},
			X2Participants: []X2Participant{
				{
					ID:             "x2-42",
					RegistrationID: "reg-42",
					CarNumber:      "99",
				},
			},
		}

		MapToX2(mapper)

		result := mapper.Results["live-2"]
		if result.X2 == nil || result.X2.ID != "x2-42" {
			t.Fatalf("expected registration fallback to x2-42")
		}
		if result.Strategy != MatchByRegistration {
			t.Fatalf("expected strategy %q, got %q", MatchByRegistration, result.Strategy)
		}
	})
}

func TestMapToX2_Normalization(t *testing.T) {
	t.Run("car_number_07_matches_7", func(t *testing.T) {
		mapper := &TimingToX2Mapper{
			LiveParticipants: []LiveTimingParticipant{
				{ID: "live-3", CarNumber: "07"},
			},
			X2Participants: []X2Participant{
				{ID: "x2-7", CarNumber: "7"},
			},
		}

		MapToX2(mapper)

		result := mapper.Results["live-3"]
		if result.X2 == nil || result.X2.ID != "x2-7" {
			t.Fatalf("expected car number normalization match")
		}
		if result.Strategy != MatchByCarNumber {
			t.Fatalf("expected strategy %q, got %q", MatchByCarNumber, result.Strategy)
		}
	})

	t.Run("name_and_team_normalization_fallback", func(t *testing.T) {
		mapper := &TimingToX2Mapper{
			LiveParticipants: []LiveTimingParticipant{
				{
					ID:         "live-4",
					DriverName: "  Jane   Doe ",
					TeamName:   "VDS Racing!!",
				},
			},
			X2Participants: []X2Participant{
				{
					ID:         "x2-jane",
					DriverName: "jane doe",
					TeamName:   "vds   racing",
				},
			},
		}

		MapToX2(mapper)

		result := mapper.Results["live-4"]
		if result.X2 == nil || result.X2.ID != "x2-jane" {
			t.Fatalf("expected driver+team fallback match")
		}
		if result.Strategy != MatchByDriverAndTeam {
			t.Fatalf("expected strategy %q, got %q", MatchByDriverAndTeam, result.Strategy)
		}
	})
}

func TestMapToX2_Anomalies(t *testing.T) {
	t.Run("duplicate_and_ambiguous_car_number", func(t *testing.T) {
		mapper := &TimingToX2Mapper{
			LiveParticipants: []LiveTimingParticipant{
				{ID: "live-5", CarNumber: "12"},
			},
			X2Participants: []X2Participant{
				{ID: "x2-12-a", CarNumber: "12"},
				{ID: "x2-12-b", CarNumber: "12"},
			},
		}

		MapToX2(mapper)

		result := mapper.Results["live-5"]
		if result.Strategy != NoMatch || result.X2 != nil {
			t.Fatalf("expected no match on ambiguous car number")
		}
		assertHasAnomaly(t, mapper.Anomalies, "x2_duplicate_car_number")
		assertHasAnomaly(t, mapper.Anomalies, "ambiguous_match")
		assertHasAnomaly(t, mapper.Anomalies, "x2_not_found")
	})

	t.Run("missing_car_number", func(t *testing.T) {
		mapper := &TimingToX2Mapper{
			LiveParticipants: []LiveTimingParticipant{
				{ID: "live-6", CarNumber: ""},
			},
			X2Participants: nil,
		}

		MapToX2(mapper)

		assertHasAnomaly(t, mapper.Anomalies, "live_missing_car_number")
		assertHasAnomaly(t, mapper.Anomalies, "x2_not_found")
	})

	t.Run("x2_already_mapped", func(t *testing.T) {
		mapper := &TimingToX2Mapper{
			LiveParticipants: []LiveTimingParticipant{
				{ID: "live-7-a", Transponder: "TX-777"},
				{ID: "live-7-b", Transponder: "TX-777"},
			},
			X2Participants: []X2Participant{
				{ID: "x2-777", Transponder: "tx-777"},
			},
		}

		MapToX2(mapper)

		ra := mapper.Results["live-7-a"]
		rb := mapper.Results["live-7-b"]
		if ra.Strategy != MatchByTransponder && rb.Strategy != MatchByTransponder {
			t.Fatalf("expected at least one transponder match")
		}
		if ra.Strategy != NoMatch && rb.Strategy != NoMatch {
			t.Fatalf("expected one participant to remain unmatched due to already mapped X2")
		}
		assertHasAnomaly(t, mapper.Anomalies, "x2_already_mapped")
	})
}

func TestMapToX2_Loaders(t *testing.T) {
	t.Run("load_error_sets_last_error", func(t *testing.T) {
		wantErr := errors.New("boom")
		mapper := &TimingToX2Mapper{
			LoadLiveParticipants: func() ([]LiveTimingParticipant, error) {
				return nil, wantErr
			},
		}

		MapToX2(mapper)

		if mapper.LastError == nil {
			t.Fatalf("expected LastError to be set")
		}
		if mapper.LastError.Error() == "" {
			t.Fatalf("expected LastError message")
		}
		if len(mapper.Results) != 0 {
			t.Fatalf("expected no results on load error")
		}
	})

	t.Run("callbacks_override_inline_lists", func(t *testing.T) {
		mapper := &TimingToX2Mapper{
			LiveParticipants: []LiveTimingParticipant{
				{ID: "inline-live", CarNumber: "1"},
			},
			X2Participants: []X2Participant{
				{ID: "inline-x2", CarNumber: "1"},
			},
			LoadLiveParticipants: func() ([]LiveTimingParticipant, error) {
				return []LiveTimingParticipant{
					{ID: "callback-live", CarNumber: "55"},
				}, nil
			},
			LoadX2Participants: func() ([]X2Participant, error) {
				return []X2Participant{
					{ID: "callback-x2", CarNumber: "55"},
				}, nil
			},
		}

		MapToX2(mapper)

		if _, exists := mapper.Results["inline-live"]; exists {
			t.Fatalf("did not expect inline participant to be used when callback is present")
		}
		result, exists := mapper.Results["callback-live"]
		if !exists || result.X2 == nil || result.X2.ID != "callback-x2" {
			t.Fatalf("expected callback data to be used for mapping")
		}
	})
}

func assertHasAnomaly(t *testing.T, anomalies []MappingAnomaly, anomalyType string) {
	t.Helper()
	for _, anomaly := range anomalies {
		if anomaly.Type == anomalyType {
			return
		}
	}
	t.Fatalf("expected anomaly %q, got %+v", anomalyType, anomalies)
}
