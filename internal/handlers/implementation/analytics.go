package implementation

import (
	"net/http"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/service"
)

type analyticsHandler struct {
	svc service.AnalyticsService
}

func newAnalyticsHandler(svc service.AnalyticsService) *analyticsHandler {
	return &analyticsHandler{svc: svc}
}

func (h *analyticsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if userID == "" || from == "" || to == "" {
		respondError(w, http.StatusBadRequest, "user_id, from and to are required")
		return
	}

	ctx := r.Context()
	analytics, err := h.svc.GetAnalytics(ctx, userID, from, to)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   analytics,
	})
}
