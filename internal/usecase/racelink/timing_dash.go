package racelink

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type LiveTimingParticipant struct {
	ID             string
	Transponder    string
	RegistrationID string
	CarNumber      string
	DriverName     string
	TeamName       string
}

type X2Participant struct {
	ID             string
	Transponder    string
	RegistrationID string
	CarNumber      string
	DriverName     string
	TeamName       string
}

type MatchStrategy string

const (
	MatchByTransponder   MatchStrategy = "transponder"
	MatchByRegistration  MatchStrategy = "registration_id"
	MatchByCarNumber     MatchStrategy = "car_number"
	MatchByDriverAndTeam MatchStrategy = "driver_name_team"
	NoMatch              MatchStrategy = "no_match"
)

type MappingResult struct {
	Live      LiveTimingParticipant
	X2        *X2Participant
	Strategy  MatchStrategy
	Ambiguous bool
}

type MappingAnomaly struct {
	Type      string
	LiveID    string
	X2ID      string
	Detail    string
	Criterion string
	Key       string
}

type TimingToX2Mapper struct {
	LoadLiveParticipants func() ([]LiveTimingParticipant, error)
	LoadX2Participants   func() ([]X2Participant, error)

	LiveParticipants []LiveTimingParticipant
	X2Participants   []X2Participant

	Results   map[string]MappingResult
	Anomalies []MappingAnomaly
	LastError error
	Logf      func(format string, args ...any)
}

func MapToX2(m *TimingToX2Mapper) {
	if m == nil {
		return
	}

	m.Results = make(map[string]MappingResult)
	m.Anomalies = m.Anomalies[:0]
	m.LastError = nil

	live, x2, err := loadParticipants(m)
	if err != nil {
		m.LastError = err
		m.logf("map_to_x2: load error: %v", err)
		return
	}

	live = normalizeLiveParticipants(live)
	x2 = normalizeX2Participants(x2)

	idxTransponder := make(map[string][]int)
	idxRegistration := make(map[string][]int)
	idxCarNumber := make(map[string][]int)
	idxDriverTeam := make(map[string][]int)

	for i := range x2 {
		insertIndex(idxTransponder, x2[i].Transponder, i)
		insertIndex(idxRegistration, x2[i].RegistrationID, i)
		insertIndex(idxCarNumber, x2[i].CarNumber, i)
		insertIndex(idxDriverTeam, driverTeamKey(x2[i].DriverName, x2[i].TeamName), i)
	}

	emitDuplicateAnomalies(m, idxTransponder, "x2_duplicate_transponder", "transponder")
	emitDuplicateAnomalies(m, idxRegistration, "x2_duplicate_registration", "registration_id")
	emitDuplicateAnomalies(m, idxCarNumber, "x2_duplicate_car_number", "car_number")

	matchedX2ByID := make(map[string]string)
	for i := range live {
		entry := live[i]
		liveID := ensureLiveID(entry, i)

		if entry.CarNumber == "" {
			m.addAnomaly("live_missing_car_number", liveID, "", "live participant has no car number", "car_number", "")
		}

		matched := false
		ambiguous := false

		tryMatch := func(index map[string][]int, key string, strategy MatchStrategy, criterion string) bool {
			if key == "" {
				return false
			}
			candidates := index[key]
			if len(candidates) == 0 {
				return false
			}
			if len(candidates) > 1 {
				ambiguous = true
				m.addAnomaly("ambiguous_match", liveID, "", "multiple X2 candidates", criterion, key)
				return false
			}

			x2Candidate := x2[candidates[0]]
			x2ID := ensureX2ID(x2Candidate, candidates[0])
			if previousLiveID, used := matchedX2ByID[x2ID]; used {
				m.addAnomaly("x2_already_mapped", liveID, x2ID, fmt.Sprintf("already mapped to live id %s", previousLiveID), criterion, key)
				return false
			}

			candidate := x2Candidate
			m.Results[liveID] = MappingResult{
				Live:      entry,
				X2:        &candidate,
				Strategy:  strategy,
				Ambiguous: ambiguous,
			}
			matchedX2ByID[x2ID] = liveID
			return true
		}

		matched = tryMatch(idxTransponder, entry.Transponder, MatchByTransponder, "transponder") ||
			tryMatch(idxRegistration, entry.RegistrationID, MatchByRegistration, "registration_id") ||
			tryMatch(idxCarNumber, entry.CarNumber, MatchByCarNumber, "car_number") ||
			tryMatch(idxDriverTeam, driverTeamKey(entry.DriverName, entry.TeamName), MatchByDriverAndTeam, "driver_name_team")

		if !matched {
			m.Results[liveID] = MappingResult{
				Live:      entry,
				X2:        nil,
				Strategy:  NoMatch,
				Ambiguous: ambiguous,
			}
			m.addAnomaly("x2_not_found", liveID, "", "no X2 match found", "all", "")
		}
	}

	m.logf("map_to_x2: done live=%d x2=%d mapped=%d anomalies=%d", len(live), len(x2), mappedCount(m.Results), len(m.Anomalies))
}

func loadParticipants(m *TimingToX2Mapper) ([]LiveTimingParticipant, []X2Participant, error) {
	live := m.LiveParticipants
	x2 := m.X2Participants

	if m.LoadLiveParticipants != nil {
		items, err := m.LoadLiveParticipants()
		if err != nil {
			return nil, nil, fmt.Errorf("load live participants: %w", err)
		}
		live = items
	}

	if m.LoadX2Participants != nil {
		items, err := m.LoadX2Participants()
		if err != nil {
			return nil, nil, fmt.Errorf("load x2 participants: %w", err)
		}
		x2 = items
	}

	return live, x2, nil
}

func normalizeLiveParticipants(items []LiveTimingParticipant) []LiveTimingParticipant {
	out := make([]LiveTimingParticipant, 0, len(items))
	for _, item := range items {
		out = append(out, LiveTimingParticipant{
			ID:             normalizeID(item.ID),
			Transponder:    normalizeToken(item.Transponder),
			RegistrationID: normalizeToken(item.RegistrationID),
			CarNumber:      normalizeCarNumber(item.CarNumber),
			DriverName:     normalizeName(item.DriverName),
			TeamName:       normalizeName(item.TeamName),
		})
	}
	return out
}

func normalizeX2Participants(items []X2Participant) []X2Participant {
	out := make([]X2Participant, 0, len(items))
	for _, item := range items {
		out = append(out, X2Participant{
			ID:             normalizeID(item.ID),
			Transponder:    normalizeToken(item.Transponder),
			RegistrationID: normalizeToken(item.RegistrationID),
			CarNumber:      normalizeCarNumber(item.CarNumber),
			DriverName:     normalizeName(item.DriverName),
			TeamName:       normalizeName(item.TeamName),
		})
	}
	return out
}

func normalizeID(v string) string {
	return strings.TrimSpace(v)
}

func normalizeToken(v string) string {
	return strings.ToUpper(strings.TrimSpace(v))
}

func normalizeName(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	var b strings.Builder
	lastSpace := false
	for _, r := range v {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteRune(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func normalizeCarNumber(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range v {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	cleaned := b.String()
	if cleaned == "" {
		return ""
	}
	if onlyDigits(cleaned) {
		n, err := strconv.Atoi(cleaned)
		if err == nil {
			return strconv.Itoa(n)
		}
	}
	return cleaned
}

func onlyDigits(v string) bool {
	if v == "" {
		return false
	}
	for _, r := range v {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func driverTeamKey(driverName, teamName string) string {
	if driverName == "" && teamName == "" {
		return ""
	}
	return driverName + "|" + teamName
}

func insertIndex(index map[string][]int, key string, value int) {
	if key == "" {
		return
	}
	index[key] = append(index[key], value)
}

func emitDuplicateAnomalies(m *TimingToX2Mapper, index map[string][]int, anomalyType, criterion string) {
	for key, candidates := range index {
		if len(candidates) <= 1 {
			continue
		}
		m.addAnomaly(anomalyType, "", "", fmt.Sprintf("%d entries share the same key", len(candidates)), criterion, key)
	}
}

func ensureLiveID(entry LiveTimingParticipant, fallback int) string {
	if entry.ID != "" {
		return entry.ID
	}
	if entry.RegistrationID != "" {
		return "live:reg:" + entry.RegistrationID
	}
	if entry.Transponder != "" {
		return "live:transp:" + entry.Transponder
	}
	if entry.CarNumber != "" {
		return "live:car:" + entry.CarNumber
	}
	return fmt.Sprintf("live:index:%d", fallback)
}

func ensureX2ID(entry X2Participant, fallback int) string {
	if entry.ID != "" {
		return entry.ID
	}
	if entry.RegistrationID != "" {
		return "x2:reg:" + entry.RegistrationID
	}
	if entry.Transponder != "" {
		return "x2:transp:" + entry.Transponder
	}
	if entry.CarNumber != "" {
		return "x2:car:" + entry.CarNumber
	}
	return fmt.Sprintf("x2:index:%d", fallback)
}

func mappedCount(results map[string]MappingResult) int {
	count := 0
	for _, result := range results {
		if result.X2 != nil {
			count++
		}
	}
	return count
}

func (m *TimingToX2Mapper) addAnomaly(anomalyType, liveID, x2ID, detail, criterion, key string) {
	m.Anomalies = append(m.Anomalies, MappingAnomaly{
		Type:      anomalyType,
		LiveID:    liveID,
		X2ID:      x2ID,
		Detail:    detail,
		Criterion: criterion,
		Key:       key,
	})
	m.logf("map_to_x2 anomaly type=%s live=%s x2=%s criterion=%s key=%s detail=%s", anomalyType, liveID, x2ID, criterion, key, detail)
}

func (m *TimingToX2Mapper) logf(format string, args ...any) {
	if m.Logf != nil {
		m.Logf(format, args...)
	}
}
