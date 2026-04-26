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
	userID, from, to, ok := h.parseAnalyticsQuery(w, r)
	if !ok {
		return
	}

	observability.RecordAnalyticsRequest()

	analytics, err := h.svc.GetAnalytics(r.Context(), userID, from, to)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, analytics)
}

func (h *analyticsHandler) GetStreamAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, from, to, ok := h.parseAnalyticsQuery(w, r)
	if !ok {
		return
	}

	observability.RecordAnalyticsRequest()

	analytics, err := h.svc.GetStreamAnalytics(r.Context(), userID, from, to)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, analytics)
}

func (h *analyticsHandler) parseAnalyticsQuery(w http.ResponseWriter, r *http.Request) (userID, from, to string, ok bool) {
	userID, ok = authenticatedUserID(w, r)
	if !ok {
		return "", "", "", false
	}
	from = r.URL.Query().Get("from")
	to = r.URL.Query().Get("to")
	if !queryUserMatchesPrincipal(w, r, userID) {
		return "", "", "", false
	}
	if from == "" || to == "" {
		respondError(w, http.StatusBadRequest, "from and to are required")
		return "", "", "", false
	}
	return userID, from, to, true
}
