package transactionevents

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	EventTypeTransactionCreated = "transaction.created"
	EventTypeTransactionUpdated = "transaction.updated"
	EventTypeTransactionDeleted = "transaction.deleted"
	EventTypeStatusChanged      = "transaction.status_changed"
	SupportedSchemaVersion      = 1
	EventSourceDiplomaApp       = "diploma-app"
	DefaultTransactionsTopic    = "transactions.events"
)

// Envelope is the stable JSON contract for transaction lifecycle events.
type Envelope struct {
	EventID       string              `json:"event_id"`
	EventType     string              `json:"event_type"`
	EventTime     time.Time           `json:"event_time"`
	CorrelationID string              `json:"correlation_id"`
	SchemaVersion int                 `json:"schema_version"`
	Source        string              `json:"source"`
	AggregateID   string              `json:"aggregate_id"`
	Transaction   *TransactionPayload `json:"transaction,omitempty"`
	Before        *TransactionPayload `json:"before,omitempty"`
	After         *TransactionPayload `json:"after,omitempty"`
	OldStatus     string              `json:"old_status,omitempty"`
	NewStatus     string              `json:"new_status,omitempty"`
}

// TransactionPayload is the integration payload carried by transaction events.
// It intentionally mirrors only the externally shared event contract, not the
// internal write-side persistence model.
type TransactionPayload struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	Amount        string     `json:"amount"`
	Currency      string     `json:"currency"`
	FromAccountID *string    `json:"from_account_id"`
	ToAccountID   *string    `json:"to_account_id"`
	ProviderID    *string    `json:"provider_id"`
	CategoryID    *string    `json:"category_id"`
	Type          string     `json:"type"`
	Status        string     `json:"status"`
	Description   *string    `json:"description"`
	ExternalID    *string    `json:"external_id"`
	OccurredAt    time.Time  `json:"occurred_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at"`
}

func NewCreated(tx *TransactionPayload, eventTime time.Time) (*Envelope, error) {
	if tx == nil {
		return nil, errors.New("transaction is nil")
	}
	return New(EventTypeTransactionCreated, tx.ID, tx, nil, nil, "", "", eventTime)
}

func NewUpdated(before, after *TransactionPayload, eventTime time.Time) (*Envelope, error) {
	if after == nil {
		return nil, errors.New("transaction after is nil")
	}
	return New(EventTypeTransactionUpdated, after.ID, after, before, after, "", "", eventTime)
}

func NewDeleted(tx *TransactionPayload, eventTime time.Time) (*Envelope, error) {
	if tx == nil {
		return nil, errors.New("transaction is nil")
	}
	return New(EventTypeTransactionDeleted, tx.ID, tx, tx, nil, "", "", eventTime)
}

func NewStatusChanged(before, after *TransactionPayload, eventTime time.Time) (*Envelope, error) {
	if before == nil {
		return nil, errors.New("transaction before is nil")
	}
	if after == nil {
		return nil, errors.New("transaction after is nil")
	}
	return New(EventTypeStatusChanged, after.ID, after, before, after, before.Status, after.Status, eventTime)
}

func New(
	eventType string,
	aggregateID string,
	transaction *TransactionPayload,
	before *TransactionPayload,
	after *TransactionPayload,
	oldStatus string,
	newStatus string,
	eventTime time.Time,
) (*Envelope, error) {
	if eventTime.IsZero() {
		eventTime = time.Now().UTC()
	}

	eventID, err := newEventID()
	if err != nil {
		return nil, err
	}

	env := &Envelope{
		EventID:       eventID,
		EventType:     eventType,
		EventTime:     eventTime.UTC(),
		CorrelationID: aggregateID,
		SchemaVersion: SupportedSchemaVersion,
		Source:        EventSourceDiplomaApp,
		AggregateID:   aggregateID,
		Transaction:   transaction,
		Before:        before,
		After:         after,
		OldStatus:     oldStatus,
		NewStatus:     newStatus,
	}
	if err := env.Validate(); err != nil {
		return nil, err
	}
	return env, nil
}

func MarshalCreated(tx *TransactionPayload, eventTime time.Time) ([]byte, error) {
	env, err := NewCreated(tx, eventTime)
	if err != nil {
		return nil, err
	}
	return Marshal(env)
}

func Marshal(env *Envelope) ([]byte, error) {
	if err := env.Validate(); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("marshal %s event: %w", env.EventType, err)
	}
	return payload, nil
}

func Parse(data []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	if err := env.Validate(); err != nil {
		return nil, err
	}
	return &env, nil
}

func ParseCreated(data []byte) (*Envelope, error) {
	env, err := Parse(data)
	if err != nil {
		return nil, err
	}
	if env.EventType != EventTypeTransactionCreated {
		return nil, fmt.Errorf("unsupported event_type %q (want %q)", env.EventType, EventTypeTransactionCreated)
	}
	return env, nil
}

func TransactionForEvent(env *Envelope) *TransactionPayload {
	if env == nil {
		return nil
	}
	if env.After != nil {
		return env.After
	}
	if env.Transaction != nil {
		return env.Transaction
	}
	return env.Before
}

func (e *Envelope) Validate() error {
	if e == nil {
		return errors.New("envelope is nil")
	}
	if e.EventID == "" {
		return errors.New("event_id is empty")
	}
	if !isSupportedEventType(e.EventType) {
		return fmt.Errorf("unsupported event_type %q", e.EventType)
	}
	if e.SchemaVersion != SupportedSchemaVersion {
		return fmt.Errorf("unsupported schema_version %d (want %d)", e.SchemaVersion, SupportedSchemaVersion)
	}
	if e.AggregateID == "" {
		if e.Transaction != nil && e.Transaction.ID != "" {
			e.AggregateID = e.Transaction.ID
		} else {
			return errors.New("aggregate_id is empty")
		}
	}
	if e.CorrelationID == "" {
		e.CorrelationID = e.AggregateID
	}

	switch e.EventType {
	case EventTypeTransactionCreated:
		if err := validateTransactionMatchesAggregate(e.Transaction, e.AggregateID, "transaction"); err != nil {
			return err
		}
	case EventTypeTransactionUpdated:
		if err := validateTransactionMatchesAggregate(e.After, e.AggregateID, "after"); err != nil {
			return err
		}
	case EventTypeTransactionDeleted:
		if err := validateTransactionMatchesAggregate(e.Transaction, e.AggregateID, "transaction"); err != nil {
			return err
		}
	case EventTypeStatusChanged:
		if err := validateTransactionMatchesAggregate(e.Before, e.AggregateID, "before"); err != nil {
			return err
		}
		if err := validateTransactionMatchesAggregate(e.After, e.AggregateID, "after"); err != nil {
			return err
		}
		if e.OldStatus == "" || e.NewStatus == "" {
			return errors.New("status_changed event requires old_status and new_status")
		}
	}
	return nil
}

func isSupportedEventType(eventType string) bool {
	switch eventType {
	case EventTypeTransactionCreated, EventTypeTransactionUpdated, EventTypeTransactionDeleted, EventTypeStatusChanged:
		return true
	default:
		return false
	}
}

func validateTransactionMatchesAggregate(tx *TransactionPayload, aggregateID, field string) error {
	if tx == nil {
		return fmt.Errorf("%s is nil", field)
	}
	if tx.ID == "" {
		return fmt.Errorf("%s.id is empty", field)
	}
	if tx.ID != aggregateID {
		return fmt.Errorf("%s.id %q does not match aggregate_id %q", field, tx.ID, aggregateID)
	}
	return nil
}

func newEventID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate event_id: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	), nil
}
