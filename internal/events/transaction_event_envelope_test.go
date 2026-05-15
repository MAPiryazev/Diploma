package events

import (
	transactionevents "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/contracts/transactionevents"
	"strings"
	"testing"
	"time"
)

func TestNewTransactionStatusChangedEnvelope_SetsStatusesAndCorrelation(t *testing.T) {
	before := &transactionevents.TransactionPayload{ID: "tx-1", UserID: "user-1", Status: "pending"}
	after := &transactionevents.TransactionPayload{ID: "tx-1", UserID: "user-1", Status: "done"}

	env, err := transactionevents.NewStatusChanged(before, after, time.Time{})
	if err != nil {
		t.Fatalf("NewStatusChanged() error = %v", err)
	}

	if env.EventType != EventTypeStatusChanged {
		t.Fatalf("EventType = %q, want %q", env.EventType, EventTypeStatusChanged)
	}
	if env.AggregateID != "tx-1" || env.CorrelationID != "tx-1" {
		t.Fatalf("aggregate/correlation = %q/%q", env.AggregateID, env.CorrelationID)
	}
	if env.OldStatus != "pending" || env.NewStatus != "done" {
		t.Fatalf("statuses = %q -> %q", env.OldStatus, env.NewStatus)
	}
	if env.EventID == "" {
		t.Fatal("expected generated event id")
	}
	if env.EventTime.IsZero() {
		t.Fatal("expected generated event time")
	}
}

func TestParseTransactionCreatedJSONRejectsWrongEventType(t *testing.T) {
	raw := `{
		"event_id":"e1",
		"event_type":"transaction.updated",
		"event_time":"2025-01-01T12:00:00Z",
		"correlation_id":"tx-1",
		"schema_version":1,
		"source":"diploma-app",
		"aggregate_id":"tx-1",
		"transaction":{
			"id":"tx-1",
			"user_id":"user-1",
			"amount":"10.00",
			"currency":"USD",
			"type":"expense",
			"status":"done",
			"occurred_at":"2025-01-01T12:00:00Z",
			"created_at":"2025-01-01T12:00:00Z",
			"updated_at":"2025-01-01T12:00:00Z"
		},
		"before":{
			"id":"tx-1",
			"user_id":"user-1",
			"amount":"10.00",
			"currency":"USD",
			"type":"expense",
			"status":"pending",
			"occurred_at":"2025-01-01T12:00:00Z",
			"created_at":"2025-01-01T12:00:00Z",
			"updated_at":"2025-01-01T12:00:00Z"
		},
		"after":{
			"id":"tx-1",
			"user_id":"user-1",
			"amount":"10.00",
			"currency":"USD",
			"type":"expense",
			"status":"done",
			"occurred_at":"2025-01-01T12:00:00Z",
			"created_at":"2025-01-01T12:00:00Z",
			"updated_at":"2025-01-01T12:00:00Z"
		}
	}`

	_, err := ParseTransactionCreatedJSON([]byte(raw))
	if err == nil {
		t.Fatal("expected error for non-created event")
	}
	if !strings.Contains(err.Error(), `want "transaction.created"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserIDKeyFromTransactionEventFallsBackToAfterAndBefore(t *testing.T) {
	tests := []struct {
		name string
		env  *TransactionEventEnvelope
		want string
	}{
		{
			name: "after fallback",
			env: &TransactionEventEnvelope{
				After: &transactionevents.TransactionPayload{ID: "tx-1", UserID: "user-after"},
			},
			want: "user-after",
		},
		{
			name: "before fallback",
			env: &TransactionEventEnvelope{
				Before: &transactionevents.TransactionPayload{ID: "tx-1", UserID: "user-before"},
			},
			want: "user-before",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := userIDKeyFromTransactionEvent(tt.env)
			if err != nil {
				t.Fatalf("userIDKeyFromTransactionEvent() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("userIDKeyFromTransactionEvent() = %q, want %q", got, tt.want)
			}
		})
	}
}
