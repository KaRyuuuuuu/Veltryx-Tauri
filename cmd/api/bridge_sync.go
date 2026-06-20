package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const bridgeSharedKeyEnv = "BRIDGE_SHARED_KEY_B64"

type bridgeEncryptedEnvelope struct {
	Version    int    `json:"version"`
	AgentID    string `json:"agentId"`
	SentAt     string `json:"sentAt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type bridgeFlagState struct {
	Code  int    `json:"code"`
	Label string `json:"label"`
}

type bridgeSectorState struct {
	ID   int             `json:"id"`
	Name string          `json:"name"`
	Flag bridgeFlagState `json:"flag"`
}

type bridgeFlagsPayload struct {
	Schema     string              `json:"schema"`
	Source     string              `json:"source"`
	User       string              `json:"user"`
	SentAt     string              `json:"sentAt"`
	Connection map[string]any      `json:"connection"`
	GlobalFlag bridgeFlagState     `json:"globalFlag"`
	Sectors    []bridgeSectorState `json:"sectors"`
}

type bridgeStateStore struct {
	mu          sync.RWMutex
	LastAgentID string             `json:"lastAgentId"`
	LastSeenAt  string             `json:"lastSeenAt"`
	Payload     bridgeFlagsPayload `json:"payload"`
}

func (s *bridgeStateStore) Save(agentID string, payload bridgeFlagsPayload) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastAgentID = agentID
	s.LastSeenAt = time.Now().UTC().Format(time.RFC3339)
	s.Payload = payload
}

func (s *bridgeStateStore) Snapshot() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.LastSeenAt == "" {
		return map[string]any{
			"ok":      true,
			"present": false,
		}
	}

	return map[string]any{
		"ok":          true,
		"present":     true,
		"lastAgentId": s.LastAgentID,
		"lastSeenAt":  s.LastSeenAt,
		"payload":     s.Payload,
	}
}

func loadBridgeSharedKey() ([]byte, error) {
	raw := os.Getenv(bridgeSharedKeyEnv)
	if raw == "" {
		return nil, fmt.Errorf("%s is not set", bridgeSharedKeyEnv)
	}

	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", bridgeSharedKeyEnv, err)
	}

	switch len(key) {
	case 16, 24, 32:
		return key, nil
	default:
		return nil, fmt.Errorf("%s must decode to 16, 24 or 32 bytes", bridgeSharedKeyEnv)
	}
}

func decryptBridgePayload(env bridgeEncryptedEnvelope) (bridgeFlagsPayload, error) {
	var payload bridgeFlagsPayload

	key, err := loadBridgeSharedKey()
	if err != nil {
		return payload, err
	}

	nonce, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil {
		return payload, fmt.Errorf("decode nonce: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		return payload, fmt.Errorf("decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return payload, fmt.Errorf("new cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return payload, fmt.Errorf("new gcm: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return payload, fmt.Errorf("decrypt payload: %w", err)
	}

	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return payload, fmt.Errorf("unmarshal payload: %w", err)
	}

	return payload, nil
}

func newBridgeUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(_ *http.Request) bool {
			// Desktop app + local browser dev need permissive origin handling here.
			return true
		},
	}
}
