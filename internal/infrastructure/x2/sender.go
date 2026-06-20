package x2

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

func (c *Client) SendRaw(messageType int, payload []byte) error {
	if c.conn == nil {
		return fmt.Errorf("websocket non initialisee")
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := c.conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return fmt.Errorf("set websocket write deadline: %w", err)
	}
	if err := c.conn.WriteMessage(messageType, payload); err != nil {
		return fmt.Errorf("write websocket message: %w", err)
	}
	if err := c.conn.SetWriteDeadline(time.Time{}); err != nil {
		return fmt.Errorf("reset websocket write deadline: %w", err)
	}

	return nil
}

// SendJSON serialize n'importe quelle struct Go en JSON puis l'envoie sur le websocket.
//
// Exemple:
//
//	err := client.SendJSON(models.AuthenticateRequestEnvelope{
//		Authenticate: models.AuthenticateRequest{
//			ClientName: "X2Link API test",
//			User:       "admin",
//			Hash:       "sha256(adminHash + token)",
//		},
//	})
//
// Pour choisir ce que tu envoies, cree ou reutilise une struct dans internal/domain/models
// avec les bons tags json, puis passe-la a SendJSON.
func (c *Client) SendJSON(payload any) error {
	message, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal websocket payload: %w", err)
	}

	return c.SendRaw(websocket.TextMessage, message)
}
