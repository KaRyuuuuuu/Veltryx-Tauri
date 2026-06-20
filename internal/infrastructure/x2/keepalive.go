package x2

import (
	"veltryx/internal/domain/models"
)

// keepAlive repond au serveur X2 pour maintenir la session active
func (c *Client) keepAlive() {
	if err := c.SendJSON(models.KeepAliveEnvelope{Msg: 37}); err != nil {
		return
	}
}

// sendSubscribe envoie une requete d'abonnement aux canaux specifies
func (c *Client) sendSubscribe(channels ...string) error {
	return c.SendJSON(models.SubscribeEnvelope{
		Subscribe: channels,
	})
}
