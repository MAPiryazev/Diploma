package authjwt

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestGenerateAndParse(t *testing.T) {
	now := time.Unix(1700000000, 0)
	token, claims, err := Generate("secret", "user-1", "operator", "jti-1", time.Hour, now)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if claims.ID != "jti-1" {
		t.Fatalf("claims.ID = %q, want jti-1", claims.ID)
	}

	parsed, err := Parse("secret", token, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if parsed.UserID != "user-1" || parsed.Subject != "user-1" || parsed.Role != "operator" {
		t.Fatalf("claims = %+v", parsed)
	}
}

func TestParseRejectsInvalidSignature(t *testing.T) {
	now := time.Unix(1700000000, 0)
	token, _, err := Generate("secret", "user-1", "operator", "jti-1", time.Hour, now)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	_, err = Parse("other-secret", token, now)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Parse() error = %v, want ErrInvalid", err)
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	now := time.Unix(1700000000, 0)
	token, _, err := Generate("secret", "user-1", "operator", "jti-1", time.Second, now)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	_, err = Parse("secret", token, now.Add(2*time.Second))
	if !errors.Is(err, ErrExpired) {
		t.Fatalf("Parse() error = %v, want ErrExpired", err)
	}
}

func TestParseRequiresJWTShape(t *testing.T) {
	_, err := Parse("secret", strings.Repeat("x", 32), time.Now())
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Parse() error = %v, want ErrInvalid", err)
	}
}
