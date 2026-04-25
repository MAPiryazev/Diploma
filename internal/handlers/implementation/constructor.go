package implementation

import (
	"context"
	"net/http"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/events"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/handlers"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/repository"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/service"
	"github.com/wb-go/wbf/dbpg"
)

type healthHandler struct {
	db *dbpg.DB
}

func newHealthHandler(db *dbpg.DB) *healthHandler {
	return &healthHandler{db: db}
}

func (h *healthHandler) Health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *healthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	h.writeReady(w, r)
}

func (h *healthHandler) writeReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Master.PingContext(ctx); err != nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status":   "error",
			"database": "down",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"database": "up",
	})
}

type handlersImpl struct {
	transaction handlers.TransactionHandler
	analytics   handlers.AnalyticsHandler
	health      handlers.HealthHandler
}

func (h *handlersImpl) Transaction() handlers.TransactionHandler {
	return h.transaction
}

func (h *handlersImpl) Analytics() handlers.AnalyticsHandler {
	return h.analytics
}

func (h *handlersImpl) Health() handlers.HealthHandler {
	return h.health
}

func NewHandlers(svcs *service.Services, repos *repository.Repositories, database *dbpg.DB, publisher events.Publisher) handlers.Handlers {
	return &handlersImpl{
		transaction: newTransactionHandler(svcs.Transaction, repos.Idempotency, repos.Audit, publisher),
		analytics:   newAnalyticsHandler(svcs.Analytics),
		health:      newHealthHandler(database),
	}
}
