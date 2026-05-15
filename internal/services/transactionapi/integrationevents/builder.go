package integrationevents

import (
	"fmt"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/repository"
	transactionevents "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/contracts/transactionevents"
)

type Builder struct{}

func NewBuilder() Builder {
	return Builder{}
}

func (Builder) BuildCreated(tx *models.Transaction) ([]repository.OutboxMessage, error) {
	payload := payloadFromModel(tx)
	env, err := transactionevents.NewCreated(payload, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	message, err := marshalOutboxMessage(env, tx.UserID)
	if err != nil {
		return nil, err
	}
	return []repository.OutboxMessage{message}, nil
}

func (Builder) BuildUpdated(before, after *models.Transaction) ([]repository.OutboxMessage, error) {
	beforePayload := payloadFromModel(before)
	afterPayload := payloadFromModel(after)

	updatedEnv, err := transactionevents.NewUpdated(beforePayload, afterPayload, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	updatedMessage, err := marshalOutboxMessage(updatedEnv, after.UserID)
	if err != nil {
		return nil, err
	}

	messages := []repository.OutboxMessage{updatedMessage}
	if before.Status == after.Status {
		return messages, nil
	}

	statusChangedEnv, err := transactionevents.NewStatusChanged(beforePayload, afterPayload, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	statusChangedMessage, err := marshalOutboxMessage(statusChangedEnv, after.UserID)
	if err != nil {
		return nil, err
	}
	return append(messages, statusChangedMessage), nil
}

func (Builder) BuildDeleted(tx *models.Transaction) ([]repository.OutboxMessage, error) {
	payload := payloadFromModel(tx)
	env, err := transactionevents.NewDeleted(payload, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	message, err := marshalOutboxMessage(env, tx.UserID)
	if err != nil {
		return nil, err
	}
	return []repository.OutboxMessage{message}, nil
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

func marshalOutboxMessage(env *transactionevents.Envelope, messageKey string) (repository.OutboxMessage, error) {
	payload, err := transactionevents.Marshal(env)
	if err != nil {
		return repository.OutboxMessage{}, fmt.Errorf("marshal %s outbox event: %w", env.EventType, err)
	}

	return repository.OutboxMessage{
		EventID:     env.EventID,
		AggregateID: env.AggregateID,
		EventType:   env.EventType,
		MessageKey:  messageKey,
		Payload:     payload,
	}, nil
}
