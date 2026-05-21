package runtime

import (
	"context"
	"errors"
	"log"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/config"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/repository"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/security"
	processingsubscriber "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/processing/subscriber"
	transactionintegration "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/transactionapi/integrationevents"
	transactionevents "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/contracts/transactionevents"
	platformruntime "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/platform/runtime"
)

const (
	serviceName    = "transaction-executor"
	subscriberName = "transaction-executor"
)

type transactionResolver interface {
	ResolvePending(ctx context.Context, id string, approved bool) (*models.Transaction, bool, error)
}

func Run(envPath string) error {
	cfg, err := platformruntime.LoadConfig(envPath)
	if err != nil {
		return err
	}

	return processingsubscriber.Run(envPath, processingsubscriber.Options{
		ServiceName:    serviceName,
		SubscriberName: subscriberName,
		NeedsLedgerDB:  true,
	}, newExecutorHandler(cfg))
}

type executorHandler struct {
	txRepo transactionResolver
}

func newExecutorHandler(cfg *config.Config) processingsubscriber.HandlerFactory {
	return func(deps processingsubscriber.Dependencies) processingsubscriber.Handler {
		return executorHandler{
			txRepo: repository.NewTransactionRepository(deps.LedgerDB, transactionintegration.NewBuilder(), cfg.Kafka.Topic),
		}
	}
}

func (h executorHandler) Handle(ctx context.Context, env *transactionevents.Envelope) error {
	if env == nil {
		return errors.New("event envelope is nil")
	}

	approved, ok := decisionFromEventType(env.EventType)
	if !ok {
		return nil
	}
	if h.txRepo == nil {
		return errors.New("transaction resolver is nil")
	}

	tx := transactionevents.TransactionForEvent(env)
	if tx == nil {
		return errors.New("transaction payload is nil")
	}
	if tx.DeletedAt != nil {
		return nil
	}

	resolved, changed, err := h.txRepo.ResolvePending(ctx, tx.ID, approved)
	if err != nil {
		return err
	}

	finalStatus := tx.Status
	if resolved != nil {
		finalStatus = resolved.Status
	}

	log.Printf(
		"transaction decision executed decision_event_id=%s tx_id=%s approved=%t changed=%t final_status=%s",
		security.MaskID(env.EventID),
		security.MaskID(tx.ID),
		approved,
		changed,
		finalStatus,
	)
	return nil
}

func decisionFromEventType(eventType string) (bool, bool) {
	switch eventType {
	case transactionevents.EventTypeTransactionApproved:
		return true, true
	case transactionevents.EventTypeTransactionRejected:
		return false, true
	default:
		return false, false
	}
}
