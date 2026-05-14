package implementation

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/errors"
)

func TestHandleServiceError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantInBody string
	}{
		{
			name:       "validation error struct",
			err:        &apperrors.ValidationError{Field: "x", Message: "bad"},
			wantStatus: http.StatusUnprocessableEntity,
			wantInBody: "x: bad",
		},
		{name: "not found", err: apperrors.ErrNotFound, wantStatus: http.StatusNotFound, wantInBody: "not found"},
		{name: "conflict", err: apperrors.ErrConflict, wantStatus: http.StatusConflict, wantInBody: "conflict"},
		{name: "already exists", err: apperrors.ErrAlreadyExists, wantStatus: http.StatusConflict, wantInBody: "already exists"},
		{name: "validation sentinel", err: apperrors.ErrValidation, wantStatus: http.StatusUnprocessableEntity, wantInBody: "validation error"},
		{name: "bad request", err: apperrors.ErrBadRequest, wantStatus: http.StatusBadRequest, wantInBody: "bad request"},
		{name: "unauthorized", err: apperrors.ErrUnauthorized, wantStatus: http.StatusUnauthorized, wantInBody: "unauthorized"},
		{name: "forbidden", err: apperrors.ErrForbidden, wantStatus: http.StatusForbidden, wantInBody: "forbidden"},
		{name: "unavailable", err: apperrors.ErrUnavailable, wantStatus: http.StatusServiceUnavailable, wantInBody: "service unavailable"},
		{name: "unknown", err: errors.New("db exploded"), wantStatus: http.StatusInternalServerError, wantInBody: "internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handleServiceError(rec, tt.err)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			body := rec.Body.String()
			if !strings.Contains(body, tt.wantInBody) {
				t.Fatalf("body %q does not contain %q", body, tt.wantInBody)
			}
			var payload map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("invalid json: %v", err)
			}
			if payload["status"] != "error" {
				t.Fatalf("status field = %v", payload["status"])
			}
		})
	}
}
