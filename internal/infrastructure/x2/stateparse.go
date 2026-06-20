package x2

import (
	"encoding/json"

	"veltryx/internal/domain/models"
)

func (c *Client) handleOperationalPayload(msg []byte) bool {
	var payload map[string]any
	if err := json.Unmarshal(msg, &payload); err != nil {
		return false
	}

	handled := false

	if raw, ok := payload["racelink"]; ok {
		if items := parseRaceLinks(raw); len(items) > 0 {
			c.setRaceLinks(items)
			handled = true
		}
	}
	if raw, ok := payload["activelist"]; ok {
		if items := parseActiveList(raw); len(items) > 0 {
			c.setRaceLinks(items)
			handled = true
		}
	}
	if raw, ok := payload["registration"]; ok {
		if items := parseRegistrations(raw); len(items) > 0 {
			c.setRaceLinks(items)
			handled = true
		}
	}
	if raw, ok := payload["baselinklist"]; ok {
		if items := parseBaseLinks(raw); len(items) > 0 {
			c.setBaseLinks(items)
			handled = true
		}
	}
	if raw, ok := payload["sector"]; ok {
		if items := parseSectors(raw); len(items) > 0 {
			c.setSectorStates(items)
			handled = true
		}
	}
	if raw, ok := payload["globalflag"]; ok {
		if flag := parseGlobalFlag(raw); flag >= 0 {
			c.setGlobalFlag(flag)
			handled = true
		}
	}
	if raw, ok := payload["alerts"]; ok {
		if items := parseAlerts(raw); len(items) > 0 {
			c.setAlerts(items)
			handled = true
		}
	}

	return handled
}

func parseRaceLinks(raw any) []models.RaceLink {
	container, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	list, ok := container["list"].([]any)
	if !ok {
		return nil
	}
	out := make([]models.RaceLink, 0, len(list))
	for _, item := range list {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, models.RaceLink{
			ID:        firstInt(obj["id"], obj["racelink"]),
			Name:      firstString(obj["name"]),
			Flag:      firstInt(obj["flag"]),
			Battery:   firstInt(obj["battery"]),
			RSSI:      firstInt(obj["rssi"]),
			Status:    firstString(obj["status"]),
			BaseLink:  firstInt(obj["bid"], obj["baselink"]),
			LastSeen:  int64(firstInt(obj["lastseen"])),
			CarNumber: firstInt(obj["car"], obj["number"]),
		})
	}
	return out
}

func parseActiveList(raw any) []models.RaceLink {
	container, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	list, ok := container["list"].([]any)
	if !ok {
		return nil
	}
	out := make([]models.RaceLink, 0, len(list))
	for _, item := range list {
		switch typed := item.(type) {
		case map[string]any:
			out = append(out, models.RaceLink{
				ID:       firstInt(typed["id"], typed["racelink"]),
				RSSI:     firstInt(typed["rssi"]),
				BaseLink: firstInt(typed["bid"], typed["baselink"]),
				Status:   firstString(typed["status"]),
			})
		default:
			id := toInt(item)
			if id != 0 {
				out = append(out, models.RaceLink{ID: id, Status: "active"})
			}
		}
	}
	return out
}

func parseRegistrations(raw any) []models.RaceLink {
	container, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	list, ok := container["list"].([]any)
	if !ok {
		return nil
	}
	out := make([]models.RaceLink, 0, len(list))
	for _, item := range list {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := firstInt(obj["racelink"], obj["id"])
		if id == 0 {
			continue
		}
		out = append(out, models.RaceLink{
			ID:        id,
			CarNumber: firstInt(obj["car"], obj["number"]),
			Name:      firstString(obj["name"], obj["firstname"], obj["lastname"]),
		})
	}
	return out
}

func parseBaseLinks(raw any) []models.BaseLink {
	container, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	list, ok := container["list"].([]any)
	if !ok {
		return nil
	}
	out := make([]models.BaseLink, 0, len(list))
	for _, item := range list {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, models.BaseLink{
			ID:       firstInt(obj["id"]),
			Name:     firstString(obj["name"]),
			Hostname: firstString(obj["hostname"]),
			IPv4Addr: firstString(obj["ipv4addr"], obj["host"]),
			IPv4Port: firstInt(obj["ipv4port"], obj["port"]),
			Status:   firstString(obj["status"]),
		})
	}
	return out
}

func parseSectors(raw any) []models.SectorState {
	container, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	if list, ok := container["list"].([]any); ok {
		out := make([]models.SectorState, 0, len(list))
		for index, item := range list {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name := firstString(obj["name"])
			if name == "" {
				name = "S"
			}
			out = append(out, models.SectorState{
				ID:   firstInt(obj["id"]),
				Name: name,
				Flag: firstInt(obj["flag"]),
			})
			if out[len(out)-1].ID == 0 {
				out[len(out)-1].ID = index + 1
			}
		}
		return out
	}
	id := firstInt(container["id"])
	if id == 0 {
		return nil
	}
	return []models.SectorState{{
		ID:   id,
		Name: firstString(container["name"]),
		Flag: firstInt(container["flag"]),
	}}
}

func parseGlobalFlag(raw any) int {
	container, ok := raw.(map[string]any)
	if !ok {
		return -1
	}
	return firstInt(container["flag"])
}

func parseAlerts(raw any) []models.AlertMessage {
	switch typed := raw.(type) {
	case []any:
		out := make([]models.AlertMessage, 0, len(typed))
		for _, item := range typed {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			out = append(out, models.AlertMessage{
				Code:    firstString(obj["code"], obj["id"]),
				Message: firstString(obj["message"], obj["text"], obj["name"]),
				Level:   firstString(obj["level"], obj["severity"]),
			})
		}
		return out
	case map[string]any:
		if list, ok := typed["list"]; ok {
			return parseAlerts(list)
		}
		if msg := firstString(typed["message"], typed["text"]); msg != "" {
			return []models.AlertMessage{{
				Code:    firstString(typed["code"], typed["id"]),
				Message: msg,
				Level:   firstString(typed["level"], typed["severity"]),
			}}
		}
	}
	return nil
}

func firstInt(values ...any) int {
	for _, value := range values {
		if out := toInt(value); out != 0 {
			return out
		}
	}
	return 0
}

func firstString(values ...any) string {
	for _, value := range values {
		if out := toString(value); out != "" {
			return out
		}
	}
	return ""
}
