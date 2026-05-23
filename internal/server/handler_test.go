package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestIsLocalRequest(t *testing.T) {
	req := httptest.NewRequest("POST", "/shutdown", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	if !isLocalRequest(req) {
		t.Fatal("127.0.0.1 should be local")
	}

	req.RemoteAddr = "192.168.1.20:12345"
	if isLocalRequest(req) {
		t.Fatal("LAN address should not be local")
	}
}

func TestTextPreviewUsesRunes(t *testing.T) {
	content := strings.Repeat("传", 81) + "🚀"
	preview := textPreview(content, 80)
	if !utf8.ValidString(preview) {
		t.Fatal("preview is not valid UTF-8")
	}
	if got := len([]rune(preview)); got != 81 {
		t.Fatalf("preview rune length = %d, want 81", got)
	}
	if !strings.HasSuffix(preview, "…") {
		t.Fatalf("preview = %q, want ellipsis suffix", preview)
	}
}

func TestContentDispositionSupportsUTF8Filename(t *testing.T) {
	header := contentDisposition(`报告"2026".txt`)
	if !strings.Contains(header, `filename="___2026_.txt"`) {
		t.Fatalf("Content-Disposition fallback = %q", header)
	}
	if !strings.Contains(header, `filename*=UTF-8''%E6%8A%A5%E5%91%8A%222026%22.txt`) {
		t.Fatalf("Content-Disposition UTF-8 filename missing: %q", header)
	}
}

func TestCorsMiddlewareAllowsQuickShareOrigins(t *testing.T) {
	s := &Server{lanIP: "192.168.1.10", port: 8080}
	handler := s.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest("GET", "/files", nil)
	req.Header.Set("Origin", "http://127.0.0.1:8080")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:8080" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestCorsMiddlewareRejectsUnknownPreflightOrigin(t *testing.T) {
	s := &Server{lanIP: "192.168.1.10", port: 8080}
	handler := s.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for rejected preflight")
	}))

	req := httptest.NewRequest("OPTIONS", "/upload", nil)
	req.Header.Set("Origin", "http://example.test")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestCorsMiddlewareRejectsUnknownPostOrigin(t *testing.T) {
	s := &Server{lanIP: "192.168.1.10", port: 8080}
	handler := s.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for rejected POST")
	}))

	req := httptest.NewRequest("POST", "/share-text", strings.NewReader(`{"text":"hello"}`))
	req.Header.Set("Origin", "http://example.test")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}
