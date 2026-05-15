package runtime

import (
	"context"
	"errors"
	"log"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/observability"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/security"
	processingstore "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/processing/store"
	processingsubscriber "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/processing/subscriber"
	transactionevents "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/contracts/transactionevents"
	"github.com/wb-go/wbf/dbpg"
)

const (
	serviceName    = "projection-builder"
	subscriberName = "projection-builder"
)

func Run(envPath string) error {
	return processingsubscriber.Run(envPath, processingsubscriber.Options{
		ServiceName:    serviceName,
		SubscriberName: subscriberName,
	}, newProjectionHandler)
}

type projectionHandler struct {
	statsStore                 processingstore.TransactionEventStatsStore
	analyticsTransactionsStore processingstore.AnalyticsTransactionsStore
}

func newProjectionHandler(database *dbpg.DB) processingsubscriber.Handler {
	return projectionHandler{
		statsStore:                 processingstore.NewPostgresTransactionEventStatsStore(database),
		analyticsTransactionsStore: processingstore.NewPostgresAnalyticsTransactionsStore(database),
	}
}

func (h projectionHandler) Handle(ctx context.Context, env *transactionevents.Envelope) error {
	tx := transactionevents.TransactionForEvent(env)
	if tx == nil {
		return errors.New("transaction payload is nil")
	}

	log.Printf(
		"%s applied event_id=%s tx_id=%s user_id=%s amount=%s currency=%s type=%s status=%s",
		env.EventType,
		security.MaskID(env.EventID),
		security.MaskID(env.AggregateID),
		security.MaskID(tx.UserID),
		security.MaskAmount(tx.Amount),
		tx.Currency,
		tx.Type,
		tx.Status,
	)

	if h.statsStore != nil {
		if err := h.statsStore.Apply(ctx, buildTransactionEventStat(env, tx)); err != nil {
			return err
		}
	}

	if h.analyticsTransactionsStore != nil {
		if err := h.analyticsTransactionsStore.Apply(ctx, buildAnalyticsTransaction(tx)); err != nil {
			return err
		}
	}

	observability.RecordTransactionProjectionApplied(env.EventType)
	return nil
}

func buildTransactionEventStat(env *transactionevents.Envelope, tx *transactionevents.TransactionPayload) processingstore.TransactionEventStat {
	stat := processingstore.TransactionEventStat{
		UserID:    tx.UserID,
		Currency:  tx.Currency,
		StatDate:  tx.OccurredAt,
		EventTime: env.EventTime,
	}

	switch env.EventType {
	case transactionevents.EventTypeTransactionCreated:
		stat.CreatedCount = 1
		stat.CreatedAmount = tx.Amount
	case transactionevents.EventTypeTransactionUpdated:
		stat.UpdatedCount = 1
	case transactionevents.EventTypeTransactionDeleted:
		stat.DeletedCount = 1
	case transactionevents.EventTypeStatusChanged:
		stat.StatusChangedCount = 1
	}

	return stat
}

func buildAnalyticsTransaction(tx *transactionevents.TransactionPayload) processingstore.AnalyticsTransaction {
	return processingstore.AnalyticsTransaction{
		TransactionID: tx.ID,
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
