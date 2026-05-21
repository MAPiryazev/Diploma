package runtime

import (
	"context"
	"testing"
	"time"

	transactionevents "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/contracts/transactionevents"
)

type stubRiskRepository struct {
	configs         map[string]ruleConfig
	count           int
	sum             float64
	sameAmountCount int
}

func (s stubRiskRepository) LoadRuleConfigs(context.Context) (map[string]ruleConfig, error) {
	return s.configs, nil
}

func (s stubRiskRepository) CountTransactionsSince(context.Context, string, string, time.Time, time.Time) (int, error) {
	return s.count, nil
}

func (s stubRiskRepository) SumTransactionsSince(context.Context, string, string, time.Time, time.Time) (float64, error) {
	return s.sum, nil
}

func (s stubRiskRepository) CountTransactionsByAmountSince(context.Context, string, string, string, time.Time, time.Time) (int, error) {
	return s.sameAmountCount, nil
}

func TestEvaluateRuleLargeAmount(t *testing.T) {
	handler := riskHandler{}
	tx := sampleRiskTransaction("150000.00", time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC))
	rule := defaultRuleConfigs(100000)[largeAmountRule]

	match, ok, err := handler.evaluateRule(context.Background(), rule, tx, 150000, tx.OccurredAt)
	if err != nil {
		t.Fatalf("evaluateRule() error = %v", err)
	}
	if !ok {
		t.Fatal("expected large_amount match")
	}
	if match.RuleCode != largeAmountRule {
		t.Fatalf("match.RuleCode = %q", match.RuleCode)
	}
}

func TestEvaluateRuleVelocity1H(t *testing.T) {
	handler := riskHandler{riskRepo: stubRiskRepository{count: 5}}
	tx := sampleRiskTransaction("1200.00", time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC))
	rule := defaultRuleConfigs(100000)[velocity1hRule]

	match, ok, err := handler.evaluateRule(context.Background(), rule, tx, 1200, tx.OccurredAt)
	if err != nil {
		t.Fatalf("evaluateRule() error = %v", err)
	}
	if !ok {
		t.Fatal("expected velocity_1h match")
	}
	if match.RuleCode != velocity1hRule {
		t.Fatalf("match.RuleCode = %q", match.RuleCode)
	}
}

func TestEvaluateRuleNightActivity(t *testing.T) {
	handler := riskHandler{}
	occurredAt := time.Date(2026, 5, 15, 2, 15, 0, 0, time.UTC)
	tx := sampleRiskTransaction("25000.00", occurredAt)
	rule := defaultRuleConfigs(100000)[nightActivityRule]

	match, ok, err := handler.evaluateRule(context.Background(), rule, tx, 25000, occurredAt)
	if err != nil {
		t.Fatalf("evaluateRule() error = %v", err)
	}
	if !ok {
		t.Fatal("expected night_activity match")
	}
	if match.RuleCode != nightActivityRule {
		t.Fatalf("match.RuleCode = %q", match.RuleCode)
	}
}

func TestEvaluateRuleRepeatedAmount24H(t *testing.T) {
	handler := riskHandler{riskRepo: stubRiskRepository{sameAmountCount: 2}}
	tx := sampleRiskTransaction("50000.00", time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC))
	rule := defaultRuleConfigs(100000)[repeatedAmount24hRule]

	match, ok, err := handler.evaluateRule(context.Background(), rule, tx, 50000, tx.OccurredAt)
	if err != nil {
		t.Fatalf("evaluateRule() error = %v", err)
	}
	if !ok {
		t.Fatal("expected repeated_amount_24h match")
	}
	if match.RuleCode != repeatedAmount24hRule {
		t.Fatalf("match.RuleCode = %q", match.RuleCode)
	}
}

func sampleRiskTransaction(amount string, occurredAt time.Time) *transactionevents.TransactionPayload {
	return &transactionevents.TransactionPayload{
		ID:         "aaaaaaaa-1111-1111-1111-111111111111",
		UserID:     "11111111-1111-1111-1111-111111111111",
		Amount:     amount,
		Currency:   "RUB",
		Type:       "expense",
		Status:     "done",
		OccurredAt: occurredAt,
		CreatedAt:  occurredAt,
		UpdatedAt:  occurredAt,
	}
}
