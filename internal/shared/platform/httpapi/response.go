package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	apperrors "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/errors"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/middleware"
)

func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, map[string]any{
		"status":  "error",
		"message": message,
	})
}

func RespondSuccess(w http.ResponseWriter, status int, data any) {
	RespondJSON(w, status, map[string]any{
		"status": "success",
		"data":   data,
	})
}

func HandleServiceError(w http.ResponseWriter, err error) {
	var valErr *apperrors.ValidationError
	if errors.As(err, &valErr) {
		RespondJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"status":  "error",
			"message": valErr.Error(),
		})
		return
	}

	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		RespondError(w, http.StatusNotFound, "not found")
	case errors.Is(err, apperrors.ErrConflict):
		RespondError(w, http.StatusConflict, "conflict")
	case errors.Is(err, apperrors.ErrAlreadyExists):
		RespondError(w, http.StatusConflict, "already exists")
	case errors.Is(err, apperrors.ErrValidation):
		RespondError(w, http.StatusUnprocessableEntity, "validation error")
	case errors.Is(err, apperrors.ErrBadRequest):
		RespondError(w, http.StatusBadRequest, "bad request")
	case errors.Is(err, apperrors.ErrUnauthorized):
		RespondError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, apperrors.ErrForbidden):
		RespondError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, apperrors.ErrUnavailable):
		RespondError(w, http.StatusServiceUnavailable, "service unavailable")
	default:
		RespondError(w, http.StatusInternalServerError, "internal server error")
	}
}

func AuthenticatedUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	principal, ok := middleware.GetPrincipal(r.Context())
	if !ok || principal.UserID == "" {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	return principal.UserID, true
}

func QueryUserMatchesPrincipal(w http.ResponseWriter, r *http.Request, userID string) bool {
	queryUserID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if queryUserID == "" || queryUserID == userID {
		return true
	}
	RespondError(w, http.StatusForbidden, "user_id does not match authenticated principal")
	return false
}
