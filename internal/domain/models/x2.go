package models

type X2Envelope struct {
	ClientID     int                   `json:"clientid,omitempty"`
	Msg          int                   `json:"msg,omitempty"`
	RequestID    int                   `json:"requestid,omitempty"`
	Time         int64                 `json:"time,omitempty"`
	Welcome      *WelcomePayload       `json:"welcome,omitempty"`
	Authenticate *AuthenticateResponse `json:"authenticate,omitempty"`
}

type WelcomePayload struct {
	ClientID           int    `json:"clientid"`
	ExpirationDate     string `json:"expirationdate"`
	ExpirationDaysLeft int    `json:"expirationdaysleft"`
	ProtocolVersion    int    `json:"protocolversion"`
	Token              string `json:"token"`
}

type AuthenticateRequestEnvelope struct {
	Authenticate AuthenticateRequest `json:"authenticate"`
}

type AuthenticateRequest struct {
	ClientName string `json:"clientname"`
	User       string `json:"user"`
	Hash       string `json:"hash"`
}

type AuthenticateResponse struct {
	Authenticated bool   `json:"authenticated"`
	ClientName    string `json:"clientname"`
	Role          string `json:"role"`
	User          string `json:"user"`
}

type SubscribeEnvelope struct {
	Subscribe []string `json:"subscribe"`
}

type KeepAliveEnvelope struct {
	Msg int `json:"msg"`
}

type TrackConfigurationRequestEnvelope struct {
	TrackConfiguration map[string]string `json:"trackconfiguration"`
}

type TrackConfiguration struct {
	Version  int           `json:"version"`
	Name     string        `json:"name"`
	Creator  string        `json:"creator"`
	Modified string        `json:"modified"`
	Paths    []TrackPath   `json:"paths"`
	Lines    []TrackLine   `json:"lines"`
	Sectors  []TrackSector `json:"sectors"`
}

type TrackPath struct {
	ID          int         `json:"id"`
	Name        string      `json:"name"`
	Width       float64     `json:"width"`
	Type        string      `json:"type"`
	Closed      bool        `json:"closed"`
	Coordinates [][]float64 `json:"coordinates"`
}

type TrackLine struct {
	ID    int       `json:"id"`
	Name  string    `json:"name"`
	Start []float64 `json:"start"`
	End   []float64 `json:"end"`
	Type  string    `json:"type"`
}

type TrackSector struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	Coordinate []float64 `json:"coordinate"`
	Path       int       `json:"path"`
}

type GPSPoint struct {
	RacelinkID int     `json:"racelinkId"`
	Lat        float64 `json:"lat"`
	Lon        float64 `json:"lon"`
	Speed      float64 `json:"speed,omitempty"`
	Timestamp  int64   `json:"timestamp,omitempty"`
	Vehicle    string  `json:"vehicle,omitempty"`
}

type RaceLink struct {
	ID        int    `json:"id"`
	Name      string `json:"name,omitempty"`
	CarNumber int    `json:"carNumber,omitempty"`
	Flag      int    `json:"flag,omitempty"`
	Battery   int    `json:"battery,omitempty"`
	RSSI      int    `json:"rssi,omitempty"`
	Status    string `json:"status,omitempty"`
	BaseLink  int    `json:"baseLink,omitempty"`
	LastSeen  int64  `json:"lastSeen,omitempty"`
}

type BaseLink struct {
	ID       int    `json:"id"`
	Name     string `json:"name,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	IPv4Addr string `json:"ipv4addr,omitempty"`
	IPv4Port int    `json:"ipv4port,omitempty"`
	Status   string `json:"status,omitempty"`
}

type SectorState struct {
	ID   int    `json:"id"`
	Name string `json:"name,omitempty"`
	Flag int    `json:"flag,omitempty"`
}

type AlertMessage struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Level   string `json:"level,omitempty"`
}

type CanMessage struct {
	ID    int   `json:"id"`
	CanID int   `json:"canid"`
	Data  []int `json:"data"`
}
