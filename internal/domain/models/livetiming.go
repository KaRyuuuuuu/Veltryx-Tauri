package models

import "time"

type LiveTimingSessionInfo struct {
	Name        string `json:"name"`
	Series      string `json:"series"`
	SessionType string `json:"sessionType"`
	Date        string `json:"date"`
	Time        string `json:"time"`
	Circuit     string `json:"circuit"`
	DurationMs  int64  `json:"durationMs"`
}

type LiveTimingParticipant struct {
	Number string `json:"number"`
	Team   string `json:"team"`
	Car    string `json:"car"`
	Class  string `json:"class"`
}

type LiveTimingClassification struct {
	CarNumber string `json:"carNumber"`
	Laps      int    `json:"laps"`
	PassageMs int64  `json:"passageMs"`
	LastLapMs int64  `json:"lastLapMs"`
	BestLapMs int64  `json:"bestLapMs"`
	Sector1Ms int64  `json:"sector1Ms"`
	Sector2Ms int64  `json:"sector2Ms"`
	Sector3Ms int64  `json:"sector3Ms"`
	GapMs     int64  `json:"gapMs"`
	Status    int    `json:"status"`
	PitState  int    `json:"pitState"`
	Position  int    `json:"position"`
}

type LiveTimingRaceControlMessage struct {
	Tag  string `json:"tag"`
	Code string `json:"code"`
	Text string `json:"text"`
}

type LiveTimingLeaderboardEntry struct {
	Position  int    `json:"position"`
	CarNumber string `json:"carNumber"`
	Team      string `json:"team"`
	Car       string `json:"car"`
	Class     string `json:"class"`
	Laps      int    `json:"laps"`
	PassageMs int64  `json:"passageMs"`
	LastLapMs int64  `json:"lastLapMs"`
	BestLapMs int64  `json:"bestLapMs"`
	Sector1Ms int64  `json:"sector1Ms"`
	Sector2Ms int64  `json:"sector2Ms"`
	Sector3Ms int64  `json:"sector3Ms"`
	GapMs     int64  `json:"gapMs"`
	Status    int    `json:"status"`
	PitState  int    `json:"pitState"`
}

type LiveTimingStreamSnapshot struct {
	Address          string    `json:"address"`
	Connected        bool      `json:"connected"`
	LastError        string    `json:"lastError"`
	LastChunkAt      time.Time `json:"lastChunkAt"`
	BytesReceived    int64     `json:"bytesReceived"`
	MessagesReceived int64     `json:"messagesReceived"`
	LastMessage      string    `json:"lastMessage"`
	RecentMessages   []string  `json:"recentMessages"`
}

type LiveTimingSnapshot struct {
	Session      LiveTimingSessionInfo         `json:"session"`
	Stream       LiveTimingStreamSnapshot      `json:"stream"`
	Leaderboard  []LiveTimingLeaderboardEntry  `json:"leaderboard"`
	LastMessage  *LiveTimingRaceControlMessage `json:"lastMessage,omitempty"`
	Messages     []LiveTimingRaceControlMessage `json:"messages"`
	Participants int                           `json:"participants"`
}
