package events

import (
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	transactionevents "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/contracts/transactionevents"
	"testing"
	"time"
)

func TestTransactionCreatedEnvelope_Validate_ok(t *testing.T) {
	tx := &transactionevents.TransactionPayload{ID: "e1", UserID: "u1"}
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
	tx := &transactionevents.TransactionPayload{ID: "x", UserID: "u1"}
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

func TestNewTransactionDecisionEnvelopeCompatibilityWrappers(t *testing.T) {
	tx := &transactionevents.TransactionPayload{
		ID:         "tx-1",
		UserID:     "user-1",
		Status:     "pending",
		OccurredAt: time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	model := &models.Transaction{
		ID:         tx.ID,
		UserID:     tx.UserID,
		Status:     tx.Status,
		OccurredAt: tx.OccurredAt,
		CreatedAt:  tx.CreatedAt,
		UpdatedAt:  tx.UpdatedAt,
	}

	approved, err := NewTransactionApprovedEnvelope(model, time.Time{})
	if err != nil {
		t.Fatalf("NewTransactionApprovedEnvelope() error = %v", err)
	}
	if approved.EventType != EventTypeTransactionApproved {
		t.Fatalf("approved.EventType = %q", approved.EventType)
	}

	rejected, err := NewTransactionRejectedEnvelope(model, time.Time{})
	if err != nil {
		t.Fatalf("NewTransactionRejectedEnvelope() error = %v", err)
	}
	if rejected.EventType != EventTypeTransactionRejected {
		t.Fatalf("rejected.EventType = %q", rejected.EventType)
	}
}
