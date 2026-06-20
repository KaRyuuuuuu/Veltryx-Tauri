package x2

import (
	"encoding/json"
	"fmt"
	"math"

	"veltryx/internal/domain/models"
)

func (c *Client) requestTrackConfiguration() error {
	return c.SendJSON(models.TrackConfigurationRequestEnvelope{
		TrackConfiguration: map[string]string{"action": "get"},
	})
}

func (c *Client) handleTrackingPayload(msg []byte) bool {
	var payload map[string]any
	if err := json.Unmarshal(msg, &payload); err != nil {
		return false
	}

	handled := false
	if rawTrack, ok := payload["trackconfiguration"]; ok {
		if track := parseTrackConfiguration(rawTrack); track != nil {
			c.setTrack(track)
			c.appendEvent("track", "track configuration updated", "")
			handled = true
		}
	}

	for _, key := range []string{"gps", "gpspos", "gpslatest", "positions"} {
		if rawGPS, ok := payload[key]; ok {
			c.updateGPSLatestFromPayload(rawGPS)
			handled = true
		}
	}

	return handled
}

func parseTrackConfiguration(raw any) *models.TrackConfiguration {
	trackMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	info, ok := trackMap["info"]
	if !ok {
		return nil
	}

	var buf []byte
	switch typed := info.(type) {
	case string:
		buf = []byte(typed)
	default:
		var err error
		buf, err = json.Marshal(info)
		if err != nil {
			return nil
		}
	}

	var track models.TrackConfiguration
	if err := json.Unmarshal(buf, &track); err != nil {
		return nil
	}
	return &track
}

func (c *Client) updateGPSLatestFromPayload(payload any) {
	switch typed := payload.(type) {
	case []any:
		for _, item := range typed {
			c.ingestGPSPoint(item)
		}
	case map[string]any:
		// X2 often sends GPS points in a list envelope: {"gps":{"list":[...]}}
		if list, ok := typed["list"]; ok {
			if arr, ok := list.([]any); ok {
				for _, item := range arr {
					c.ingestGPSPoint(item)
				}
				return
			}
		}
		// Fallback: single-point object.
		c.ingestGPSPoint(typed)
	}
}

func (c *Client) ingestGPSPoint(raw any) {
	m, ok := raw.(map[string]any)
	if !ok {
		return
	}

	racelinkID := toInt(m["id"])
	if racelinkID == 0 {
		racelinkID = toInt(m["racelink"])
	}
	if racelinkID == 0 {
		racelinkID = toInt(m["racelinkid"])
	}
	if racelinkID == 0 {
		return
	}

	lat := toFloat(m["lat"])
	if lat == 0 {
		lat = toFloat(m["latitude"])
	}
	lon := toFloat(m["lon"])
	if lon == 0 {
		lon = toFloat(m["lng"])
	}
	if lon == 0 {
		lon = toFloat(m["longitude"])
	}
	if !isFiniteLatLon(lat, lon) {
		return
	}

	speed := toFloat(m["speed"])
	if speed == 0 {
		speed = toFloat(m["kph"])
	}
	timestamp := int64(toInt(m["timestamp"]))
	if timestamp == 0 {
		timestamp = int64(toInt(m["time"]))
	}

	c.updateGPSPoint(models.GPSPoint{
		RacelinkID: racelinkID,
		Lat:        lat,
		Lon:        lon,
		Speed:      speed,
		Timestamp:  timestamp,
		Vehicle:    toString(m["vehicle"]),
	})
}

func toInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case float32:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		i, _ := t.Int64()
		return int(i)
	default:
		return 0
	}
}

func toFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		var parsed float64
		if _, err := fmt.Sscanf(t, "%f", &parsed); err == nil {
			return parsed
		}
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return 0
	}
	return 0
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func isFiniteLatLon(lat, lon float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) {
		return false
	}
	if lat < -90 || lat > 90 {
		return false
	}
	if lon < -180 || lon > 180 {
		return false
	}
	return true
}
