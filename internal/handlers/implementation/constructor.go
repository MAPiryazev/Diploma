package implementation

import (
	"net/http"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/handlers"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/service"
)

type healthHandler struct{}

func newHealthHandler() *healthHandler {
	return &healthHandler{}
}

func (h *healthHandler) Health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
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

func NewHandlers(svcs *service.Services) handlers.Handlers {
	return &handlersImpl{
		transaction: newTransactionHandler(svcs.Transaction),
		analytics:   newAnalyticsHandler(svcs.Analytics),
		health:      newHealthHandler(),
	}
}
