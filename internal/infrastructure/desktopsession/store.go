package desktopsession

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// User représente l'identité d'un utilisateur de session
type User struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	TenantID    string   `json:"tenantId,omitempty"`
	Permissions []string `json:"permissions"`
}

// Record contient les informations détaillées d'une session enregistrée
type Record struct {
	SessionID       string    `json:"session_id"`
	DeviceID        string    `json:"device_id"`
	User            User      `json:"user"`
	IssuedAt        time.Time `json:"issued_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	ServerSignature string    `json:"server_signature"`
}

// Store gère le stockage et la validation des sessions en mémoire et sur disque
type Store struct {
	mu      sync.RWMutex
	path    string
	records map[string]Record
}

// NewStore crée et initialise un nouveau magasin de sessions
func NewStore() *Store {
	s := &Store{
		path:    defaultPath(),
		records: make(map[string]Record),
	}
	s.load()
	return s
}

// Register enregistre ou met à jour une session utilisateur
func (s *Store) Register(record Record) Record {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	record.IssuedAt = now
	// Durée de session par défaut : 12 heures
	if !record.ExpiresAt.After(now) {
		record.ExpiresAt = now.Add(12 * time.Hour)
	}
	if record.ServerSignature == "" {
		record.ServerSignature = newToken(16)
	}
	s.records[record.SessionID] = record
	s.saveLocked()
	return record
}

// Validate vérifie la validité d'une session et rafraîchit son expiration
func (s *Store) Validate(sessionID, deviceID, lastKnownUser string) (Record, bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.records[sessionID]
	if !ok {
		return Record{}, false, "session_not_found"
	}
	now := time.Now().UTC()
	// Suppression automatique si expirée
	if now.After(record.ExpiresAt) {
		delete(s.records, sessionID)
		s.saveLocked()
		return Record{}, false, "session_expired"
	}
	// Vérification de la cohérence de l'appareil
	if deviceID != "" && record.DeviceID != "" && record.DeviceID != deviceID {
		return Record{}, false, "device_mismatch"
	}
	// Vérification de la cohérence de l'utilisateur
	if lastKnownUser != "" && record.User.ID != "" && record.User.ID != lastKnownUser {
		return Record{}, false, "user_mismatch"
	}

	// Prolongation de la session (12h supplémentaires)
	record.ExpiresAt = now.Add(12 * time.Hour)
	s.records[sessionID] = record
	s.saveLocked()
	return record, true, ""
}

// Delete supprime définitivement une session
func (s *Store) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, sessionID)
	s.saveLocked()
}

// List retourne la liste de toutes les sessions actives (non expirées)
func (s *Store) List() []Record {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	out := make([]Record, 0, len(s.records))
	changed := false
	for sessionID, record := range s.records {
		if now.After(record.ExpiresAt) {
			delete(s.records, sessionID)
			changed = true
			continue
		}
		out = append(out, record)
	}
	if changed {
		s.saveLocked()
	}
	// Tri par date d'émission décroissante
	sort.Slice(out, func(i, j int) bool {
		return out[i].IssuedAt.After(out[j].IssuedAt)
	})
	return out
}

// load charge les sessions depuis le fichier JSON sur le disque
func (s *Store) load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var records []Record
	if err := json.Unmarshal(raw, &records); err != nil {
		return
	}
	now := time.Now().UTC()
	for _, record := range records {
		if now.After(record.ExpiresAt) {
			continue
		}
		s.records[record.SessionID] = record
	}
}

// saveLocked enregistre l'état actuel des sessions sur le disque (doit être appelé avec le mutex verrouillé)
func (s *Store) saveLocked() {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return
	}
	records := make([]Record, 0, len(s.records))
	for _, record := range s.records {
		records = append(records, record)
	}
	raw, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, raw, 0o600)
}

// defaultPath détermine le chemin par défaut pour le stockage des sessions
func defaultPath() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		return filepath.Join(".", "desktop-sessions.json")
	}
	return filepath.Join(base, "veltryx", "desktop-sessions.json")
}

// newToken génère une chaîne aléatoire sécurisée pour les signatures
func newToken(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		// Repli sur une signature temporelle en cas d'erreur
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}
