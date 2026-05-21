package runtime

import (
	"context"
	"testing"

	transactionevents "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/contracts/transactionevents"
)

func TestShouldIgnoreEventType(t *testing.T) {
	tests := []struct {
		eventType string
		want      bool
	}{
		{eventType: transactionevents.EventTypeTransactionApproved, want: true},
		{eventType: transactionevents.EventTypeTransactionRejected, want: true},
		{eventType: transactionevents.EventTypeTransactionUpdated, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			if got := shouldIgnoreEventType(tt.eventType); got != tt.want {
				t.Fatalf("shouldIgnoreEventType(%q) = %t, want %t", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestHandleIgnoresDecisionEvents(t *testing.T) {
	handler := projectionHandler{}

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
}
