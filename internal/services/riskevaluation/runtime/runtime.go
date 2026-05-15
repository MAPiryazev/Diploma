package runtime

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/config"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/observability"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/security"
	processingstore "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/processing/store"
	processingsubscriber "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/processing/subscriber"
	transactionevents "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/contracts/transactionevents"
	platformruntime "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/platform/runtime"
	"github.com/wb-go/wbf/dbpg"
)

const (
	serviceName     = "risk-evaluation"
	subscriberName  = "risk-evaluation"
	largeAmountRule = "large_amount"
)

func Run(envPath string) error {
	cfg, err := platformruntime.LoadConfig(envPath)
	if err != nil {
		return err
	}

	return processingsubscriber.Run(envPath, processingsubscriber.Options{
		ServiceName:    serviceName,
		SubscriberName: subscriberName,
	}, newRiskHandler(cfg))
}

type riskHandler struct {
	monitoringStore      processingstore.MonitoringEventsStore
	largeAmountThreshold float64
}

func newRiskHandler(cfg *config.Config) processingsubscriber.HandlerFactory {
	return func(database *dbpg.DB) processingsubscriber.Handler {
		return riskHandler{
			monitoringStore:      processingstore.NewPostgresMonitoringEventsStore(database),
			largeAmountThreshold: cfg.Monitoring.LargeAmountThreshold,
		}
	}
}

func (h riskHandler) Handle(ctx context.Context, env *transactionevents.Envelope) error {
	tx := transactionevents.TransactionForEvent(env)
	if tx == nil {
		return errors.New("transaction payload is nil")
	}

	if h.largeAmountThreshold <= 0 || env.EventType == transactionevents.EventTypeTransactionDeleted {
		return nil
	}

	amount, err := strconv.ParseFloat(strings.TrimSpace(tx.Amount), 64)
	if err != nil {
		return fmt.Errorf("parse transaction amount: %w", err)
	}
	if amount < h.largeAmountThreshold {
		return nil
	}

	observability.RecordMonitoringLargeAmountEvent()
	log.Printf(
		"risk rule matched rule=%s event_id=%s tx_id=%s user_id=%s threshold=%.2f",
		largeAmountRule,
		security.MaskID(env.EventID),
		security.MaskID(tx.ID),
		security.MaskID(tx.UserID),
		h.largeAmountThreshold,
	)

	eventTime := env.EventTime
	if eventTime.IsZero() {
		eventTime = tx.OccurredAt
	}

	if h.monitoringStore == nil {
		return nil
	}

	return h.monitoringStore.Save(ctx, processingstore.MonitoringEvent{
		TransactionID: tx.ID,
		UserID:        tx.UserID,
		RuleCode:      largeAmountRule,
		Severity:      "warning",
		Reason:        "transaction amount exceeded configured threshold",
		EventTime:     eventTime,
	})
}
