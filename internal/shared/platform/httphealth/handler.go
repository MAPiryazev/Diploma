package httphealth

import (
	"context"
	"net/http"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/platform/httpapi"
	"github.com/wb-go/wbf/dbpg"
)

type Handler struct {
	db *dbpg.DB
}

func New(database *dbpg.DB) *Handler {
	return &Handler{db: database}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	httpapi.RespondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if h.db == nil || h.db.Master == nil {
		httpapi.RespondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status":   "error",
			"database": "down",
		})
		return
	}
	if err := h.db.Master.PingContext(ctx); err != nil {
		httpapi.RespondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status":   "error",
			"database": "down",
		})
		return
	}

	httpapi.RespondJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"database": "up",
	})
}
