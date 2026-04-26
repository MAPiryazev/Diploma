package events

import (
	"testing"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
)

func TestTransactionCreatedEnvelope_Validate_ok(t *testing.T) {
	tx := &models.Transaction{ID: "e1", UserID: "u1"}
	env := TransactionCreatedEnvelope{
		EventID:       "e1",
		EventType:     EventTypeTransactionCreated,
		EventTime:     time.Now().UTC(),
		CorrelationID: "tx1",
		SchemaVersion: SupportedSchemaVersion,
		Source:        "test",
		AggregateID:   "e1",
		Transaction:   tx,
	}
	if err := env.Validate(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestTransactionCreatedEnvelope_Validate_mismatchedIDs(t *testing.T) {
	tx := &models.Transaction{ID: "x", UserID: "u1"}
	env := TransactionCreatedEnvelope{
		EventID:       "e1",
		EventType:     EventTypeTransactionCreated,
		SchemaVersion: SupportedSchemaVersion,
		AggregateID:   "e1",
		Transaction:   tx,
	}
	if err := env.Validate(); err == nil {
		t.Fatal("expected error for mismatched ids")
	}
}

func TestParseTransactionCreatedJSON(t *testing.T) {
	raw := `{
		"event_id":"e1",
		"event_type":"transaction.created",
		"event_time":"2025-01-01T12:00:00Z",
		"correlation_id":"e1",
		"schema_version":1,
		"source":"diploma-app",
		"aggregate_id":"e1",
		"transaction":{
			"id":"e1",
			"user_id":"u1",
			"amount":"10.00",
			"currency":"USD",
			"type":"expense",
			"status":"posted",
			"occurred_at":"2025-01-01T12:00:00Z",
			"created_at":"2025-01-01T12:00:00Z",
			"updated_at":"2025-01-01T12:00:00Z"
		}
	}`
	env, err := ParseTransactionCreatedJSON([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if env.Transaction.UserID != "u1" {
		t.Fatalf("user_id: got %q", env.Transaction.UserID)
	}
}
