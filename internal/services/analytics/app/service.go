package app

import (
	"context"
	"fmt"

	apperrors "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/errors"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/validator"
)

type Repository interface {
	GetSum(ctx context.Context, userID string, from, to string) (string, error)
	GetAvg(ctx context.Context, userID string, from, to string) (string, error)
	GetCount(ctx context.Context, userID string, from, to string) (int64, error)
	GetMedian(ctx context.Context, userID string, from, to string) (string, error)
	GetPercentile90(ctx context.Context, userID string, from, to string) (string, error)
	GetStreamStats(ctx context.Context, userID string, from, to string) ([]StreamAnalyticsRow, error)
	ListMonitoringAlerts(ctx context.Context, userID string, filter MonitoringAlertsFilter) ([]MonitoringAlert, error)
}

type Service interface {
	GetAnalytics(ctx context.Context, userID, from, to string) (*AnalyticsResponse, error)
	GetStreamAnalytics(ctx context.Context, userID, from, to string) (*StreamAnalyticsResponse, error)
	GetMonitoringAlerts(ctx context.Context, userID string, filter MonitoringAlertsFilter) (*MonitoringAlertsResponse, error)
}

type AnalyticsResponse struct {
	Sum          string `json:"sum"`
	Avg          string `json:"avg"`
	Count        int64  `json:"count"`
	Median       string `json:"median"`
	Percentile90 string `json:"percentile_90"`
}

type StreamAnalyticsResponse struct {
	Rows   []StreamAnalyticsRow `json:"rows"`
	Totals StreamAnalyticsTotal `json:"totals"`
}

type StreamAnalyticsRow struct {
	StatDate           string `json:"stat_date"`
	Currency           string `json:"currency"`
	CreatedCount       int64  `json:"created_count"`
	UpdatedCount       int64  `json:"updated_count"`
	DeletedCount       int64  `json:"deleted_count"`
	StatusChangedCount int64  `json:"status_changed_count"`
	CreatedAmount      string `json:"created_amount"`
	LastEventTime      string `json:"last_event_time"`
}

type StreamAnalyticsTotal struct {
	CreatedCount       int64 `json:"created_count"`
	UpdatedCount       int64 `json:"updated_count"`
	DeletedCount       int64 `json:"deleted_count"`
	StatusChangedCount int64 `json:"status_changed_count"`
}

type MonitoringAlert struct {
	ID            string `json:"id"`
	TransactionID string `json:"transaction_id"`
	UserID        string `json:"user_id"`
	RuleCode      string `json:"rule_code"`
	Severity      string `json:"severity"`
	Reason        string `json:"reason"`
	EventTime     string `json:"event_time"`
	CreatedAt     string `json:"created_at"`
}

type MonitoringAlertsFilter struct {
	Severity string `json:"severity"`
	RuleCode string `json:"rule_code"`
	From     string `json:"from"`
	To       string `json:"to"`
	Limit    int    `json:"limit"`
}

type MonitoringAlertsResponse struct {
	Alerts []MonitoringAlert `json:"alerts"`
	Count  int               `json:"count"`
	Limit  int               `json:"limit"`
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetAnalytics(ctx context.Context, userID, from, to string) (*AnalyticsResponse, error) {
	if err := validator.ValidateUUID(userID); err != nil {
		return nil, err
	}
	if err := validator.ValidateDateRange(from, to); err != nil {
		return nil, err
	}

	sum, err := s.repo.GetSum(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get sum: %w", err)
	}
	avg, err := s.repo.GetAvg(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get avg: %w", err)
	}
	count, err := s.repo.GetCount(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get count: %w", err)
	}
	median, err := s.repo.GetMedian(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get median: %w", err)
	}
	p90, err := s.repo.GetPercentile90(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get percentile 90: %w", err)
	}

	return &AnalyticsResponse{
		Sum:          sum,
		Avg:          avg,
		Count:        count,
		Median:       median,
		Percentile90: p90,
	}, nil
}

func (s *service) GetStreamAnalytics(ctx context.Context, userID, from, to string) (*StreamAnalyticsResponse, error) {
	if err := validator.ValidateUUID(userID); err != nil {
		return nil, err
	}
	if err := validator.ValidateDateRange(from, to); err != nil {
		return nil, err
	}

	rows, err := s.repo.GetStreamStats(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream analytics: %w", err)
	}

	resp := &StreamAnalyticsResponse{
		Rows: make([]StreamAnalyticsRow, 0, len(rows)),
	}
	for _, row := range rows {
		resp.Rows = append(resp.Rows, row)
		resp.Totals.CreatedCount += row.CreatedCount
		resp.Totals.UpdatedCount += row.UpdatedCount
		resp.Totals.DeletedCount += row.DeletedCount
		resp.Totals.StatusChangedCount += row.StatusChangedCount
	}
	return resp, nil
}

func (s *service) GetMonitoringAlerts(ctx context.Context, userID string, filter MonitoringAlertsFilter) (*MonitoringAlertsResponse, error) {
	if err := validator.ValidateUUID(userID); err != nil {
		return nil, err
	}
	if filter.From != "" || filter.To != "" {
		if err := validator.ValidateDateRange(filter.From, filter.To); err != nil {
			return nil, err
		}
	}
	if filter.Severity != "" {
		switch filter.Severity {
		case "info", "warning", "critical":
		default:
			return nil, &apperrors.ValidationError{Field: "severity", Message: "must be info, warning or critical"}
		}
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	}

	alerts, err := s.repo.ListMonitoringAlerts(ctx, userID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list monitoring alerts: %w", err)
	}

	return &MonitoringAlertsResponse{
		Alerts: alerts,
		Count:  len(alerts),
		Limit:  filter.Limit,
	}, nil
}
