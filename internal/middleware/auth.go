package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/config"
)

const authorizationHeader = "Authorization"

type Principal struct {
	UserID string
	Role   string
	Token  string
}

const principalContextKey contextKey = "principal"

func BearerAuth(tokens []config.AuthToken) func(http.Handler) http.Handler {
	byToken := make(map[string]Principal, len(tokens))
	for _, token := range tokens {
		if strings.TrimSpace(token.Token) == "" || strings.TrimSpace(token.UserID) == "" {
			continue
		}
		role := strings.TrimSpace(token.Role)
		if role == "" {
			role = "operator"
		}
		byToken[token.Token] = Principal{
			UserID: token.UserID,
			Role:   role,
			Token:  token.Token,
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimSpace(r.Header.Get(authorizationHeader))
			if raw == "" {
				writeAuthError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			token, ok := strings.CutPrefix(raw, "Bearer ")
			if !ok || strings.TrimSpace(token) == "" {
				writeAuthError(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}

			principal, ok := byToken[strings.TrimSpace(token)]
			if !ok {
				writeAuthError(w, http.StatusUnauthorized, "invalid bearer token")
				return
			}

			ctx := context.WithValue(r.Context(), principalContextKey, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetPrincipal(ctx context.Context) (Principal, bool) {
	if ctx == nil {
		return Principal{}, false
	}
	principal, ok := ctx.Value(principalContextKey).(Principal)
	return principal, ok
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"status":"error","message":"` + message + `"}`))
}
