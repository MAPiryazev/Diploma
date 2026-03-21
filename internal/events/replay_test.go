package events

import (
	"context"
	"testing"
)

func TestRepublishTransactionCreatedPayload_invalidEnvelope(t *testing.T) {
	w, err := NewReplayWriter([]string{"127.0.0.1:9092"}, "transactions.events")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Close() }()

	err = RepublishTransactionCreatedPayload(context.Background(), w, []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for invalid envelope")
	}
}
