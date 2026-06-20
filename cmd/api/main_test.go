package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsAllowedOrigin(t *testing.T) {
	if !isAllowedOrigin("http://127.0.0.1:3000") {
		t.Fatalf("expected 127.0.0.1 origin to be allowed")
	}
	if !isAllowedOrigin("http://localhost:5173") {
		t.Fatalf("expected localhost origin to be allowed")
	}
	if isAllowedOrigin("https://localhost:5173") {
		t.Fatalf("expected https localhost origin to be blocked")
	}
	if isAllowedOrigin("http://example.com") {
		t.Fatalf("expected external origin to be blocked")
	}
}

func TestPermissionsFromHeader(t *testing.T) {
	got := permissionsFromHeader("a, b ,,c")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("unexpected permissions parsing: %#v", got)
	}
	if permissions := permissionsFromHeader(""); permissions != nil {
		t.Fatalf("expected nil for empty header, got %#v", permissions)
	}
}

func TestWithCORS(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := withCORS(next)

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatalf("expected next handler to be called")
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("missing allow-origin header")
	}

	called = false
	optionsReq := httptest.NewRequest(http.MethodOptions, "/x", nil)
	optionsReq.Header.Set("Origin", "http://localhost:3000")
	optionsRR := httptest.NewRecorder()
	handler.ServeHTTP(optionsRR, optionsReq)

	if called {
		t.Fatalf("did not expect next handler on preflight")
	}
	if optionsRR.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", optionsRR.Code)
	}
}
