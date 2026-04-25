package implementation

import (
	"net/http"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/observability"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/service"
)

type analyticsHandler struct {
	svc service.AnalyticsService
}

func newAnalyticsHandler(svc service.AnalyticsService) *analyticsHandler {
	return &analyticsHandler{svc: svc}
}

func (h *analyticsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if !queryUserMatchesPrincipal(w, r, userID) {
		return
	}
	if from == "" || to == "" {
		respondError(w, http.StatusBadRequest, "from and to are required")
		return
	}

	observability.RecordAnalyticsRequest()

	ctx := r.Context()
	analytics, err := h.svc.GetAnalytics(ctx, userID, from, to)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, analytics)
}
