package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
)

const (
	EventTypeTransactionCreated = "transaction.created"
	SupportedSchemaVersion      = 1
)

// TransactionCreatedEnvelope is the JSON contract for transaction.created (producer and consumer).
type TransactionCreatedEnvelope struct {
	EventID       string              `json:"event_id"`
	EventType     string              `json:"event_type"`
	EventTime     time.Time           `json:"event_time"`
	CorrelationID string              `json:"correlation_id"`
	SchemaVersion int                 `json:"schema_version"`
	Source        string              `json:"source"`
	Transaction   *models.Transaction `json:"transaction"`
}

func (e *TransactionCreatedEnvelope) Validate() error {
	if e == nil {
		return errors.New("envelope is nil")
	}
	if e.EventID == "" {
		return errors.New("event_id is empty")
	}
	if e.EventType != EventTypeTransactionCreated {
		return fmt.Errorf("unsupported event_type %q (want %q)", e.EventType, EventTypeTransactionCreated)
	}
	if e.SchemaVersion != SupportedSchemaVersion {
		return fmt.Errorf("unsupported schema_version %d (want %d)", e.SchemaVersion, SupportedSchemaVersion)
	}
	if e.Transaction == nil {
		return errors.New("transaction is nil")
	}
	if e.Transaction.ID == "" {
		return errors.New("transaction.id is empty")
	}
	if e.Transaction.ID != e.EventID {
		return fmt.Errorf("transaction.id %q does not match event_id %q", e.Transaction.ID, e.EventID)
	}
	return nil
}

// ParseTransactionCreatedJSON unmarshals and validates a transaction.created payload.
func ParseTransactionCreatedJSON(data []byte) (*TransactionCreatedEnvelope, error) {
	var env TransactionCreatedEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	if err := env.Validate(); err != nil {
		return nil, err
	}
	return &env, nil
}
