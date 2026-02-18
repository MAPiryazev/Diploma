package implementation

import (
	"encoding/json"
	"errors"
	"net/http"

	apperrors "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/errors"
)

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"status":  "error",
		"message": message,
	})
}

func handleServiceError(w http.ResponseWriter, err error) {
	var valErr *apperrors.ValidationError
	if errors.As(err, &valErr) {
		respondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"status":  "error",
			"message": valErr.Error(),
		})
		return
	}

	if errors.Is(err, apperrors.ErrNotFound) {
		respondError(w, http.StatusNotFound, "not found")
		return
	}

	if errors.Is(err, apperrors.ErrConflict) {
		respondError(w, http.StatusConflict, "conflict")
		return
	}

	if errors.Is(err, apperrors.ErrAlreadyExists) {
		respondError(w, http.StatusConflict, "already exists")
		return
	}

	if errors.Is(err, apperrors.ErrValidation) {
		respondError(w, http.StatusUnprocessableEntity, "validation error")
		return
	}

	if errors.Is(err, apperrors.ErrBadRequest) {
		respondError(w, http.StatusBadRequest, "bad request")
		return
	}

	if errors.Is(err, apperrors.ErrUnauthorized) {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if errors.Is(err, apperrors.ErrForbidden) {
		respondError(w, http.StatusForbidden, "forbidden")
		return
	}

	if errors.Is(err, apperrors.ErrUnavailable) {
		respondError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	respondError(w, http.StatusInternalServerError, "internal server error")
}
