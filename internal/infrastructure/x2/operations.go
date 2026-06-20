package x2

import "veltryx/internal/domain/models"

func (c *Client) SetGlobalFlag(flag int) error {
	return c.SendJSON(map[string]any{
		"globalflag": map[string]any{
			"flag": flag,
		},
	})
}

func (c *Client) SetSectorFlag(sectorID int, flag int) error {
	return c.SendJSON(map[string]any{
		"sector": map[string]any{
			"action": "setflag",
			"id":     sectorID,
			"flag":   flag,
		},
	})
}

func (c *Client) SendCan(message models.CanMessage) error {
	return c.SendJSON(map[string]any{
		"racelink": map[string]any{
			"action": "sendcan",
			"id":     message.ID,
			"canid":  message.CanID,
			"data":   message.Data,
		},
	})
}

func defaultSubscriptions() []string {
	return []string{
		"gps",
		"status",
		"activelist",
		"registration",
		"racelink",
		"baselink",
		"geotrigger",
		"alerts",
		"sector",
		"globalflag",
	}
}

func flagCatalog() []models.AlertMessage {
	return []models.AlertMessage{
		{Code: "1", Message: "Green", Level: "info"},
		{Code: "2", Message: "Yellow", Level: "warn"},
		{Code: "3", Message: "SC", Level: "warn"},
		{Code: "4", Message: "Red", Level: "critical"},
		{Code: "0", Message: "Clear", Level: "info"},
	}
}
