package x2

import (
	"crypto/sha256"
	"fmt"

	"veltryx/internal/domain/models"
)

// computeAuthHash calcule le hash SHA256 pour l'authentification X2
func computeAuthHash(password, token string) string {
	// Le mot de passe est d'abord haché en SHA256
	adminHash := sha256.Sum256([]byte(password))
	step := fmt.Sprintf("%x", adminHash)
	// Le résultat est ensuite haché avec le token de session
	final := sha256.Sum256([]byte(step + token))
	return fmt.Sprintf("%x", final)
}

// handleWelcome gère le message de bienvenue initial du serveur X2
func (c *Client) handleWelcome(welcome *models.WelcomePayload) {
	if welcome == nil {
		return
	}

	c.setWelcome(welcome)
	c.token = welcome.Token
	// Déclenche l'envoi des identifiants d'authentification
	_ = c.sendAuth()
}

// handleAuthOK gère la réponse positive d'authentification du serveur X2
func (c *Client) handleAuthOK(auth *models.AuthenticateResponse) {
	if auth == nil {
		return
	}

	if !auth.Authenticated {
		return
	}

	c.setAuth(auth)
	// Une fois authentifié, on passe en état 'ready'
	c.setState("ready")

	// S'abonne aux flux de données par défaut
	if err := c.sendSubscribe(defaultSubscriptions()...); err != nil {
		return
	}

	// Récupère la configuration actuelle du circuit
	_ = c.requestTrackConfiguration()
}

// sendAuth construit et envoie la requête d'authentification au serveur X2
func (c *Client) sendAuth() error {
	err := c.SendJSON(models.AuthenticateRequestEnvelope{
		Authenticate: models.AuthenticateRequest{
			ClientName: c.clientName,
			User:       c.user,
			Hash:       computeAuthHash(c.password, c.token),
		},
	})
	if err != nil {
		return fmt.Errorf("send auth message: %w", err)
	}

	return nil
}
