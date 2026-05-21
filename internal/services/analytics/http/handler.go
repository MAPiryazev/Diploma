package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/observability"
	analyticsapp "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/analytics/app"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/platform/httpapi"
)

const (
	defaultMonitoringAlertsLimit = 50
	maxMonitoringAlertsLimit     = 200
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

func (h *Handler) GetMonitoringAlerts(w http.ResponseWriter, r *http.Request) {
	userID, filter, ok := parseMonitoringAlertsQuery(w, r)
	if !ok {
		return
	}

	observability.RecordAnalyticsRequest()
	alerts, err := h.svc.GetMonitoringAlerts(r.Context(), userID, filter)
	if err != nil {
		httpapi.HandleServiceError(w, err)
		return
	}
	httpapi.RespondSuccess(w, http.StatusOK, alerts)
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

func parseMonitoringAlertsQuery(w http.ResponseWriter, r *http.Request) (string, analyticsapp.MonitoringAlertsFilter, bool) {
	userID, ok := httpapi.AuthenticatedUserID(w, r)
	if !ok {
		return "", analyticsapp.MonitoringAlertsFilter{}, false
	}
	if !httpapi.QueryUserMatchesPrincipal(w, r, userID) {
		return "", analyticsapp.MonitoringAlertsFilter{}, false
	}

	filter := analyticsapp.MonitoringAlertsFilter{
		Severity: strings.TrimSpace(r.URL.Query().Get("severity")),
		RuleCode: strings.TrimSpace(r.URL.Query().Get("rule_code")),
		From:     strings.TrimSpace(r.URL.Query().Get("from")),
		To:       strings.TrimSpace(r.URL.Query().Get("to")),
		Limit:    defaultMonitoringAlertsLimit,
	}

	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 || limit > maxMonitoringAlertsLimit {
			httpapi.RespondError(w, http.StatusBadRequest, "limit must be between 1 and 200")
			return "", analyticsapp.MonitoringAlertsFilter{}, false
		}
		filter.Limit = limit
	}

	if (filter.From == "") != (filter.To == "") {
		httpapi.RespondError(w, http.StatusBadRequest, "from and to must be provided together")
		return "", analyticsapp.MonitoringAlertsFilter{}, false
	}

	return userID, filter, true
}
