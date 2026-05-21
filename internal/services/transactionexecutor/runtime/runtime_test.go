package runtime

import (
	"context"
	"testing"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	transactionevents "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/contracts/transactionevents"
)

type recordingResolver struct {
	called   bool
	txID     string
	approved bool
	result   *models.Transaction
}

func (r *recordingResolver) ResolvePending(_ context.Context, id string, approved bool) (*models.Transaction, bool, error) {
	r.called = true
	r.txID = id
	r.approved = approved
	if r.result == nil {
		r.result = &models.Transaction{ID: id, Status: "done"}
	}
	return r.result, true, nil
}

func TestDecisionFromEventType(t *testing.T) {
	tests := []struct {
		eventType string
		wantOK    bool
		wantValue bool
	}{
		{eventType: transactionevents.EventTypeTransactionApproved, wantOK: true, wantValue: true},
		{eventType: transactionevents.EventTypeTransactionRejected, wantOK: true, wantValue: false},
		{eventType: transactionevents.EventTypeTransactionCreated, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			gotValue, gotOK := decisionFromEventType(tt.eventType)
			if gotOK != tt.wantOK {
				t.Fatalf("decisionFromEventType(%q) ok=%t want %t", tt.eventType, gotOK, tt.wantOK)
			}
			if gotValue != tt.wantValue {
				t.Fatalf("decisionFromEventType(%q) value=%t want %t", tt.eventType, gotValue, tt.wantValue)
			}
		})
	}
}

func TestHandleIgnoresNonDecisionEvents(t *testing.T) {
	resolver := &recordingResolver{}
	handler := executorHandler{txRepo: resolver}

	err := handler.Handle(context.Background(), &transactionevents.Envelope{
		EventID:   "event-1",
		EventType: transactionevents.EventTypeTransactionCreated,
		Transaction: &transactionevents.TransactionPayload{
			ID:     "tx-1",
			UserID: "user-1",
			Status: "pending",
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if resolver.called {
		t.Fatal("ResolvePending() should not be called for non-decision events")
	}
}

func TestHandleApprovedDecisionCallsResolver(t *testing.T) {
	resolver := &recordingResolver{}
	handler := executorHandler{txRepo: resolver}

	err := handler.Handle(context.Background(), &transactionevents.Envelope{
		EventID:   "event-1",
		EventType: transactionevents.EventTypeTransactionApproved,
		Transaction: &transactionevents.TransactionPayload{
			ID:     "tx-1",
			UserID: "user-1",
			Status: "pending",
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if !resolver.called {
		t.Fatal("ResolvePending() was not called")
	}
	if resolver.txID != "tx-1" {
		t.Fatalf("ResolvePending() txID = %q", resolver.txID)
	}
	if !resolver.approved {
		t.Fatal("ResolvePending() approved = false, want true")
	}
}

func TestHandleRejectedDecisionCallsResolver(t *testing.T) {
	resolver := &recordingResolver{result: &models.Transaction{ID: "tx-1", Status: "failed"}}
	handler := executorHandler{txRepo: resolver}

	err := handler.Handle(context.Background(), &transactionevents.Envelope{
		EventID:   "event-1",
		EventType: transactionevents.EventTypeTransactionRejected,
		Transaction: &transactionevents.TransactionPayload{
			ID:     "tx-1",
			UserID: "user-1",
			Status: "pending",
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if !resolver.called {
		t.Fatal("ResolvePending() was not called")
	}
	if resolver.approved {
		t.Fatal("ResolvePending() approved = true, want false")
	}
}
