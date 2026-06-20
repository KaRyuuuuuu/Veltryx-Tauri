package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"veltryx/internal/infrastructure/desktopsession"
	"veltryx/internal/infrastructure/livetiming"
	"veltryx/internal/infrastructure/x2"
)

func main() {
	loadLocalEnv()

	client := x2.NewClient("192.168.103.199", "admin", "admin", "X2Link API test")
	liveClient := livetiming.NewClient("192.168.103.35:2058")
	sessionStore := desktopsession.NewStore()

	liveClient.Start()
	defer func() {
		_ = client.Close()
	}()

	mux := newAPIMux(client, liveClient, sessionStore)
	addr := "127.0.0.1:8080"
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		panic(err)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	return strings.HasPrefix(origin, "http://127.0.0.1:") ||
		strings.HasPrefix(origin, "http://localhost:")
}

func permissionsFromHeader(raw string) []string {
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	permissions := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		permissions = append(permissions, value)
	}
	return permissions
}

func loadLocalEnv() {
	loadDotEnvFile(".env")
	loadDotEnvFile(filepath.Join("..", ".env"))
}

func loadDotEnvFile(path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		entry := strings.TrimSpace(line)
		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}

		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"")
		if key == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		_ = os.Setenv(key, value)
	}
}
