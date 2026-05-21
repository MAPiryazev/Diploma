package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/config"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/middleware"
	analyticsapp "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/analytics/app"
)

type stubAnalyticsService struct {
	lastUserID string
	lastFilter analyticsapp.MonitoringAlertsFilter
	response   *analyticsapp.MonitoringAlertsResponse
}

func (s *stubAnalyticsService) GetAnalytics(context.Context, string, string, string) (*analyticsapp.AnalyticsResponse, error) {
	return nil, nil
}

func (s *stubAnalyticsService) GetStreamAnalytics(context.Context, string, string, string) (*analyticsapp.StreamAnalyticsResponse, error) {
	return nil, nil
}

func (s *stubAnalyticsService) GetMonitoringAlerts(_ context.Context, userID string, filter analyticsapp.MonitoringAlertsFilter) (*analyticsapp.MonitoringAlertsResponse, error) {
	s.lastUserID = userID
	s.lastFilter = filter
	return s.response, nil
}

func TestGetMonitoringAlertsSuccess(t *testing.T) {
	svc := &stubAnalyticsService{
		response: &analyticsapp.MonitoringAlertsResponse{
			Alerts: []analyticsapp.MonitoringAlert{{
				ID:            "alert-1",
				TransactionID: "aaaaaaaa-1111-1111-1111-111111111111",
				UserID:        "11111111-1111-1111-1111-111111111111",
				RuleCode:      "large_amount",
				Severity:      "warning",
				Reason:        "amount exceeded threshold",
				EventTime:     "2026-05-15T12:00:00Z",
				CreatedAt:     "2026-05-15T12:00:01Z",
			}},
			Count: 1,
			Limit: 25,
		},
	}
	handler := NewHandler(svc)
	protected := middleware.JWTAuth("", []config.AuthToken{{
		Token:  "dev-token",
		UserID: "11111111-1111-1111-1111-111111111111",
		Role:   "operator",
	}})(http.HandlerFunc(handler.GetMonitoringAlerts))

	req := httptest.NewRequest(http.MethodGet, "/monitoring/alerts?severity=warning&rule_code=large_amount&limit=25", nil)
	req.Header.Set("Authorization", "Bearer dev-token")
	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if svc.lastUserID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("userID = %q", svc.lastUserID)
	}
	if svc.lastFilter.Limit != 25 || svc.lastFilter.Severity != "warning" || svc.lastFilter.RuleCode != "large_amount" {
		t.Fatalf("filter = %+v", svc.lastFilter)
	}

	var payload struct {
		Status string `json:"status"`
		Data   struct {
			Count int `json:"count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Status != "success" || payload.Data.Count != 1 {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestGetMonitoringAlertsRejectsPartialDateRange(t *testing.T) {
	svc := &stubAnalyticsService{}
	handler := NewHandler(svc)
	protected := middleware.JWTAuth("", []config.AuthToken{{
		Token:  "dev-token",
		UserID: "11111111-1111-1111-1111-111111111111",
		Role:   "operator",
	}})(http.HandlerFunc(handler.GetMonitoringAlerts))

	req := httptest.NewRequest(http.MethodGet, "/monitoring/alerts?from=2026-05-15T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer dev-token")
	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
