package authjwt

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const algorithm = "HS256"

var (
	ErrExpired          = errors.New("jwt expired")
	ErrInvalid          = errors.New("invalid jwt")
	ErrMissingPrincipal = errors.New("jwt principal is missing")
)

type Claims struct {
	Subject   string `json:"sub,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Role      string `json:"role,omitempty"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	ID        string `json:"jti,omitempty"`
}

type header struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

func Generate(secret, userID, role, tokenID string, ttl time.Duration, now time.Time) (string, Claims, error) {
	if strings.TrimSpace(secret) == "" {
		return "", Claims{}, fmt.Errorf("%w: empty secret", ErrInvalid)
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", Claims{}, ErrMissingPrincipal
	}
	role = strings.TrimSpace(role)
	if role == "" {
		role = "operator"
	}
	if ttl <= 0 {
		return "", Claims{}, fmt.Errorf("%w: ttl must be positive", ErrInvalid)
	}
	if tokenID == "" {
		var err error
		tokenID, err = randomTokenID()
		if err != nil {
			return "", Claims{}, err
		}
	}

	claims := Claims{
		Subject:   userID,
		UserID:    userID,
		Role:      role,
		ExpiresAt: now.Add(ttl).Unix(),
		IssuedAt:  now.Unix(),
		ID:        tokenID,
	}
	token, err := sign(secret, claims)
	if err != nil {
		return "", Claims{}, err
	}
	return token, claims, nil
}

func Parse(secret, token string, now time.Time) (Claims, error) {
	if strings.TrimSpace(secret) == "" {
		return Claims{}, fmt.Errorf("%w: empty secret", ErrInvalid)
	}

	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalid
	}

	var hdr header
	if err := decodeJSON(parts[0], &hdr); err != nil {
		return Claims{}, fmt.Errorf("%w: header", ErrInvalid)
	}
	if hdr.Algorithm != algorithm || hdr.Type != "JWT" {
		return Claims{}, fmt.Errorf("%w: unsupported header", ErrInvalid)
	}

	signed := parts[0] + "." + parts[1]
	expected := signature(secret, signed)
	actual, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(actual, expected) {
		return Claims{}, fmt.Errorf("%w: signature", ErrInvalid)
	}

	var claims Claims
	if err := decodeJSON(parts[1], &claims); err != nil {
		return Claims{}, fmt.Errorf("%w: claims", ErrInvalid)
	}
	if claims.UserID == "" {
		claims.UserID = claims.Subject
	}
	if strings.TrimSpace(claims.UserID) == "" {
		return Claims{}, ErrMissingPrincipal
	}
	if claims.Role == "" {
		claims.Role = "operator"
	}
	if claims.ExpiresAt <= 0 || now.Unix() >= claims.ExpiresAt {
		return Claims{}, ErrExpired
	}
	return claims, nil
}

func sign(secret string, claims Claims) (string, error) {
	headerJSON, err := json.Marshal(header{Algorithm: algorithm, Type: "JWT"})
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signed := encodedHeader + "." + encodedClaims
	encodedSignature := base64.RawURLEncoding.EncodeToString(signature(secret, signed))
	return signed + "." + encodedSignature, nil
}

func signature(secret, signed string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signed))
	return mac.Sum(nil)
}

func decodeJSON(part string, out any) error {
	raw, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func randomTokenID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
