package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/authjwt"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/config"
)

func TestJWTAuthAcceptsValidJWT(t *testing.T) {
	token, _, err := authjwt.Generate("secret", "user-1", "admin", "jti-1", time.Hour, time.Now())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	handler := JWTAuth("secret", nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := GetPrincipal(r.Context())
		if !ok {
			t.Fatal("principal not found")
		}
		if principal.UserID != "user-1" || principal.Role != "admin" {
			t.Fatalf("principal = %+v", principal)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestJWTAuthRejectsMissingAndInvalidJWT(t *testing.T) {
	handler := JWTAuth("secret", nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	for name, authHeader := range map[string]string{
		"missing": "",
		"invalid": "Bearer broken-token",
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestJWTAuthKeepsLegacyTokenFallback(t *testing.T) {
	handler := JWTAuth("secret", []config.AuthToken{{
		Token:  "dev-token",
		UserID: "user-1",
		Role:   "operator",
	}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := GetPrincipal(r.Context())
		if !ok || principal.UserID != "user-1" {
			t.Fatalf("principal = %+v ok=%v", principal, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer dev-token")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
