package x2

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

const defaultPath = "/_x2api._tcp/"

func (c *Client) Connect() error {
	c.mu.Lock()
	if c.conn != nil {
		c.mu.Unlock()
		return nil
	}
	// Canal pour signaler l'arret des goroutines associees a la connexion
	c.done = make(chan struct{})
	c.mu.Unlock()

	u := url.URL{
		Scheme: "ws",
		Host:   fmt.Sprintf("%s:23456", c.host),
		Path:   defaultPath,
	}

	// Etablissement de la connexion WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.state = "wait_welcome"
	c.connectedAt = time.Now()
	c.totalMsgs = 0
	c.lastMsg = 0
	c.lastRaw = ""
	c.lastAt = time.Time{}
	c.welcome = nil
	c.auth = nil
	c.events = c.events[:0]
	// Definit une limite de lecture pour eviter les messages trop volumineux
	c.conn.SetReadLimit(4 << 20)
	c.mu.Unlock()

	// Lance l'ecouteur de messages en arriere-plan
	go c.listen()
	// go c.startKeepAlive()
	c.PublishSnapshot()

	return nil
}

// Close ferme la connexion au serveur X2 et nettoie les ressources
func (c *Client) Close() error {
	c.mu.Lock()
	select {
	case <-c.done:
	default:
		close(c.done)
	}

	if c.conn == nil {
		c.state = "disconnected"
		c.mu.Unlock()
		c.PublishSnapshot()
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	c.state = "disconnected"
	c.connectedAt = time.Time{}
	c.mu.Unlock()
	c.PublishSnapshot()
	return err
}

func (c *Client) State() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.state
}

func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.conn != nil
}
