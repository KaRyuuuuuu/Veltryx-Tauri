package x2

import (
	"encoding/json"
	"veltryx/internal/domain/models"
)

// listen est une boucle d'ecoute qui lit les messages du WebSocket
func (c *Client) listen() {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.appendEvent("socket", "read error", err.Error())
			_ = c.Close()
			return
		}
		// Dispatch du message brut pour traitement
		c.handleMessage(message)
	}
}

// handleMessage deserialise et traite les messages X2 selon leur type
func (c *Client) handleMessage(msg []byte) {
	var envelope models.X2Envelope

	if err := json.Unmarshal(msg, &envelope); err != nil {
		c.appendEvent("parse", "invalid json payload", string(msg))
		return
	}

	// Enregistre l'enveloppe pour le monitoring/historique
	c.recordEnvelope(envelope.Msg, string(msg))

	switch envelope.Msg {
	case 37: // Message de KeepAlive
		c.appendEvent("keepalive", "keepalive received", string(msg))
		c.keepAlive()
	case 14: // Message de Bienvenue (Welcome)
		c.appendEvent("welcome", "welcome received", string(msg))
		c.handleWelcome(envelope.Welcome)
	case 22: // Reponse d'authentification
		c.appendEvent("auth", "authenticate response received", string(msg))
		c.handleAuthOK(envelope.Authenticate)
	default:
		// Traitement des autres types de donnees (tracking ou operationnel)
		if c.handleTrackingPayload(msg) {
			c.PublishSnapshot()
			return
		}
		if c.handleOperationalPayload(msg) {
			c.PublishSnapshot()
			return
		}
		c.appendEvent("message", "unhandled message received", string(msg))
	}

	c.PublishSnapshot()
}
