package events

import (
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/contracts/transactionevents"
)

const (
	EventTypeTransactionCreated = transactionevents.EventTypeTransactionCreated
	EventTypeTransactionUpdated = transactionevents.EventTypeTransactionUpdated
	EventTypeTransactionDeleted = transactionevents.EventTypeTransactionDeleted
	EventTypeStatusChanged      = transactionevents.EventTypeStatusChanged
	SupportedSchemaVersion      = transactionevents.SupportedSchemaVersion
	EventSourceDiplomaApp       = transactionevents.EventSourceDiplomaApp
	DefaultTransactionsTopic    = transactionevents.DefaultTransactionsTopic
)

// TransactionEventEnvelope is kept as a compatibility alias for runtime packages
// that still import the legacy events package during the migration.
type TransactionEventEnvelope = transactionevents.Envelope

// TransactionCreatedEnvelope is kept as an alias for older tests and tools.
type TransactionCreatedEnvelope = transactionevents.Envelope

func NewTransactionCreatedEnvelope(tx *models.Transaction, eventTime time.Time) (*TransactionEventEnvelope, error) {
	return transactionevents.NewCreated(payloadFromModel(tx), eventTime)
}

func NewTransactionUpdatedEnvelope(before, after *models.Transaction, eventTime time.Time) (*TransactionEventEnvelope, error) {
	return transactionevents.NewUpdated(payloadFromModel(before), payloadFromModel(after), eventTime)
}

func NewTransactionDeletedEnvelope(tx *models.Transaction, eventTime time.Time) (*TransactionEventEnvelope, error) {
	return transactionevents.NewDeleted(payloadFromModel(tx), eventTime)
}

func NewTransactionStatusChangedEnvelope(before, after *models.Transaction, eventTime time.Time) (*TransactionEventEnvelope, error) {
	return transactionevents.NewStatusChanged(payloadFromModel(before), payloadFromModel(after), eventTime)
}

func NewTransactionEventEnvelope(
	eventType string,
	aggregateID string,
	transaction *models.Transaction,
	before *models.Transaction,
	after *models.Transaction,
	oldStatus string,
	newStatus string,
	eventTime time.Time,
) (*TransactionEventEnvelope, error) {
	return transactionevents.New(
		eventType,
		aggregateID,
		payloadFromModel(transaction),
		payloadFromModel(before),
		payloadFromModel(after),
		oldStatus,
		newStatus,
		eventTime,
	)
}

func MarshalTransactionCreatedEnvelope(tx *models.Transaction, eventTime time.Time) ([]byte, error) {
	return transactionevents.MarshalCreated(payloadFromModel(tx), eventTime)
}

func MarshalTransactionEventEnvelope(env *TransactionEventEnvelope) ([]byte, error) {
	return transactionevents.Marshal(env)
}

func ParseTransactionEventJSON(data []byte) (*TransactionEventEnvelope, error) {
	return transactionevents.Parse(data)
}

func ParseTransactionCreatedJSON(data []byte) (*TransactionEventEnvelope, error) {
	return transactionevents.ParseCreated(data)
}

func payloadFromModel(tx *models.Transaction) *transactionevents.TransactionPayload {
	if tx == nil {
		return nil
	}

	return &transactionevents.TransactionPayload{
		ID:            tx.ID,
		UserID:        tx.UserID,
		Amount:        tx.Amount,
		Currency:      tx.Currency,
		FromAccountID: tx.FromAccountID,
		ToAccountID:   tx.ToAccountID,
		ProviderID:    tx.ProviderID,
		CategoryID:    tx.CategoryID,
		Type:          tx.Type,
		Status:        tx.Status,
		Description:   tx.Description,
		ExternalID:    tx.ExternalID,
		OccurredAt:    tx.OccurredAt,
		CreatedAt:     tx.CreatedAt,
		UpdatedAt:     tx.UpdatedAt,
		DeletedAt:     tx.DeletedAt,
	}
}
