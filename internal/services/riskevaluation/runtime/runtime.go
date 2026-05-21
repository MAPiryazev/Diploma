package runtime

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

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
	monitoringStore processingstore.MonitoringEventsStore
	riskRepo        riskRepository
	defaultRules    map[string]ruleConfig
}

func newRiskHandler(cfg *config.Config) processingsubscriber.HandlerFactory {
	return func(database *dbpg.DB) processingsubscriber.Handler {
		riskRepo := newPostgresRiskRepository(database)
		return riskHandler{
			monitoringStore: processingstore.NewPostgresMonitoringEventsStore(database),
			riskRepo:        riskRepo,
			defaultRules:    defaultRuleConfigs(cfg.Monitoring.LargeAmountThreshold),
		}
	}
}

func (h riskHandler) Handle(ctx context.Context, env *transactionevents.Envelope) error {
	tx := transactionevents.TransactionForEvent(env)
	if tx == nil {
		return errors.New("transaction payload is nil")
	}

	if env.EventType == transactionevents.EventTypeTransactionDeleted || tx.DeletedAt != nil {
		return nil
	}

	amount, err := strconv.ParseFloat(strings.TrimSpace(tx.Amount), 64)
	if err != nil {
		return fmt.Errorf("parse transaction amount: %w", err)
	}

	eventTime := env.EventTime
	if eventTime.IsZero() {
		eventTime = tx.OccurredAt
	}
	if eventTime.IsZero() {
		eventTime = tx.CreatedAt
	}
	if eventTime.IsZero() {
		eventTime = tx.UpdatedAt
	}
	if eventTime.IsZero() {
		eventTime = time.Now().UTC()
	}

	occurredAt := tx.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = eventTime
	}

	rules, err := h.activeRules(ctx)
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		return nil
	}

	for _, rule := range rules {
		match, ok, err := h.evaluateRule(ctx, rule, tx, amount, occurredAt)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		observability.RecordMonitoringRuleMatch(match.RuleCode, match.Severity)
		if match.RuleCode == largeAmountRule {
			observability.RecordMonitoringLargeAmountEvent()
		}

		log.Printf(
			"risk rule matched rule=%s severity=%s event_id=%s tx_id=%s user_id=%s reason=%q",
			match.RuleCode,
			match.Severity,
			security.MaskID(env.EventID),
			security.MaskID(tx.ID),
			security.MaskID(tx.UserID),
			match.Reason,
		)

		if h.monitoringStore == nil {
			continue
		}
		if err := h.monitoringStore.Save(ctx, processingstore.MonitoringEvent{
			TransactionID: tx.ID,
			UserID:        tx.UserID,
			RuleCode:      match.RuleCode,
			Severity:      match.Severity,
			Reason:        match.Reason,
			EventTime:     eventTime,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (h riskHandler) activeRules(ctx context.Context) ([]ruleConfig, error) {
	if h.riskRepo == nil {
		return mergedRuleConfigs(h.defaultRules, nil), nil
	}

	overrides, err := h.riskRepo.LoadRuleConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("load active risk rules: %w", err)
	}

	return mergedRuleConfigs(h.defaultRules, overrides), nil
}

func (h riskHandler) evaluateRule(
	ctx context.Context,
	rule ruleConfig,
	tx *transactionevents.TransactionPayload,
	amount float64,
	occurredAt time.Time,
) (*riskMatch, bool, error) {
	if tx == nil {
		return nil, false, errors.New("transaction payload is nil")
	}

	switch rule.RuleCode {
	case largeAmountRule:
		threshold := float64Value(rule.Parameters.ThresholdAmount, 0)
		if threshold <= 0 || amount < threshold {
			return nil, false, nil
		}
		return &riskMatch{
			RuleCode: rule.RuleCode,
			Severity: rule.Severity,
			Reason: fmt.Sprintf(
				"amount %s %s exceeded configured threshold %s",
				formatMoney(amount),
				tx.Currency,
				formatMoney(threshold),
			),
		}, true, nil
	case velocity1hRule:
		windowMinutes := intValue(rule.Parameters.WindowMinutes, 60)
		maxTransactions := intValue(rule.Parameters.MaxTransactions, 5)
		if maxTransactions <= 0 {
			return nil, false, nil
		}
		count, err := h.riskRepo.CountTransactionsSince(ctx, tx.UserID, tx.ID, windowStart(occurredAt, windowMinutes), occurredAt)
		if err != nil {
			return nil, false, err
		}
		totalCount := count + 1
		if totalCount <= maxTransactions {
			return nil, false, nil
		}
		return &riskMatch{
			RuleCode: rule.RuleCode,
			Severity: rule.Severity,
			Reason: fmt.Sprintf(
				"user reached %d transactions within %d-minute window (limit %d)",
				totalCount,
				windowMinutes,
				maxTransactions,
			),
		}, true, nil
	case velocity24hAmountRule:
		windowMinutes := intValue(rule.Parameters.WindowMinutes, 24*60)
		maxTotalAmount := float64Value(rule.Parameters.MaxTotalAmount, 0)
		if maxTotalAmount <= 0 {
			return nil, false, nil
		}
		totalAmount, err := h.riskRepo.SumTransactionsSince(ctx, tx.UserID, tx.ID, windowStart(occurredAt, windowMinutes), occurredAt)
		if err != nil {
			return nil, false, err
		}
		totalAmount += amount
		if totalAmount <= maxTotalAmount {
			return nil, false, nil
		}
		return &riskMatch{
			RuleCode: rule.RuleCode,
			Severity: rule.Severity,
			Reason: fmt.Sprintf(
				"user accumulated %s %s within %d-minute window (limit %s)",
				formatMoney(totalAmount),
				tx.Currency,
				windowMinutes,
				formatMoney(maxTotalAmount),
			),
		}, true, nil
	case nightActivityRule:
		startHour := intValue(rule.Parameters.NightStartHour, 0)
		endHour := intValue(rule.Parameters.NightEndHour, 6)
		minAmount := float64Value(rule.Parameters.MinAmount, 0)
		if amount < minAmount || !isNightHour(occurredAt.UTC().Hour(), startHour, endHour) {
			return nil, false, nil
		}
		return &riskMatch{
			RuleCode: rule.RuleCode,
			Severity: rule.Severity,
			Reason: fmt.Sprintf(
				"transaction occurred during configured night window %02d:00-%02d:00 UTC",
				startHour,
				endHour,
			),
		}, true, nil
	case roundAmountRule:
		modulo := float64Value(rule.Parameters.RoundModulo, 0)
		minAmount := float64Value(rule.Parameters.MinAmount, 0)
		if amount < minAmount || !isRoundAmount(amount, modulo) {
			return nil, false, nil
		}
		return &riskMatch{
			RuleCode: rule.RuleCode,
			Severity: rule.Severity,
			Reason: fmt.Sprintf(
				"amount %s %s is a round multiple of %s and exceeded minimum %s",
				formatMoney(amount),
				tx.Currency,
				formatMoney(modulo),
				formatMoney(minAmount),
			),
		}, true, nil
	case repeatedAmount24hRule:
		windowMinutes := intValue(rule.Parameters.WindowMinutes, 24*60)
		repeatedTransactions := intValue(rule.Parameters.RepeatedTransactions, 3)
		if repeatedTransactions <= 1 {
			return nil, false, nil
		}
		count, err := h.riskRepo.CountTransactionsByAmountSince(ctx, tx.UserID, tx.ID, tx.Amount, windowStart(occurredAt, windowMinutes), occurredAt)
		if err != nil {
			return nil, false, err
		}
		totalCount := count + 1
		if totalCount < repeatedTransactions {
			return nil, false, nil
		}
		return &riskMatch{
			RuleCode: rule.RuleCode,
			Severity: rule.Severity,
			Reason: fmt.Sprintf(
				"same amount %s %s appeared %d times within %d-minute window",
				strings.TrimSpace(tx.Amount),
				tx.Currency,
				totalCount,
				windowMinutes,
			),
		}, true, nil
	default:
		return nil, false, nil
	}
}
