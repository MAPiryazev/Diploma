package http

import (
	"net/http"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/observability"
	analyticsapp "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/analytics/app"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/platform/httpapi"
)

type Handler struct {
	svc analyticsapp.Service
}

func NewHandler(svc analyticsapp.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, from, to, ok := parseAnalyticsQuery(w, r)
	if !ok {
		return
	}

	observability.RecordAnalyticsRequest()
	analytics, err := h.svc.GetAnalytics(r.Context(), userID, from, to)
	if err != nil {
		httpapi.HandleServiceError(w, err)
		return
	}
	httpapi.RespondSuccess(w, http.StatusOK, analytics)
}

func (h *Handler) GetStreamAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, from, to, ok := parseAnalyticsQuery(w, r)
	if !ok {
		return
	}

	observability.RecordAnalyticsRequest()
	analytics, err := h.svc.GetStreamAnalytics(r.Context(), userID, from, to)
	if err != nil {
		httpapi.HandleServiceError(w, err)
		return
	}
	httpapi.RespondSuccess(w, http.StatusOK, analytics)
}

func parseAnalyticsQuery(w http.ResponseWriter, r *http.Request) (userID, from, to string, ok bool) {
	userID, ok = httpapi.AuthenticatedUserID(w, r)
	if !ok {
		return "", "", "", false
	}
	from = r.URL.Query().Get("from")
	to = r.URL.Query().Get("to")
	if !httpapi.QueryUserMatchesPrincipal(w, r, userID) {
		return "", "", "", false
	}
	if from == "" || to == "" {
		httpapi.RespondError(w, http.StatusBadRequest, "from and to are required")
		return "", "", "", false
	}
	return userID, from, to, true
}
