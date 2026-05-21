package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDPreservesIncomingHeaderAndContext(t *testing.T) {
	const wantRequestID = "req-123"

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := GetRequestID(r.Context()); got != wantRequestID {
			t.Fatalf("GetRequestID() = %q, want %q", got, wantRequestID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(RequestIDHeader, wantRequestID)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get(RequestIDHeader); got != wantRequestID {
		t.Fatalf("response %s = %q, want %q", RequestIDHeader, got, wantRequestID)
	}
}

func TestRequestIDGeneratesHeaderWhenMissing(t *testing.T) {
	var ctxRequestID string

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxRequestID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	headerRequestID := rec.Header().Get(RequestIDHeader)
	if headerRequestID == "" {
		t.Fatal("expected generated request id")
	}
	if ctxRequestID != headerRequestID {
		t.Fatalf("context request id = %q, header request id = %q", ctxRequestID, headerRequestID)
	}
}

func TestRecoveryReturnsInternalServerErrorOnPanic(t *testing.T) {
	handler := RequestID(Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != `{"status":"error","message":"internal server error"}` {
		t.Fatalf("body = %q", body)
	}
	if got := rec.Header().Get(RequestIDHeader); got == "" {
		t.Fatal("expected request id header to be preserved on panic")
	}
}
