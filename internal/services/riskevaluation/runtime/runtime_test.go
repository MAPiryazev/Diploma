package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	transactionevents "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/contracts/transactionevents"
)

type stubRiskRepository struct {
	configs         map[string]ruleConfig
	count           int
	sum             float64
	sameAmountCount int
}

type stubTransactionRepository struct {
	publishedID      string
	publishedApprove bool
}

func (s *stubTransactionRepository) PublishDecision(_ context.Context, tx *models.Transaction, approved bool) error {
	if tx != nil {
		s.publishedID = tx.ID
	}
	s.publishedApprove = approved
	return nil
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

func TestShouldRejectSeverity(t *testing.T) {
	tests := []struct {
		severity string
		want     bool
	}{
		{severity: "info", want: false},
		{severity: "warning", want: true},
		{severity: "critical", want: true},
		{severity: " WARNING ", want: true},
	}

	for _, tt := range tests {
		if got := shouldRejectSeverity(tt.severity); got != tt.want {
			t.Fatalf("shouldRejectSeverity(%q) = %t, want %t", tt.severity, got, tt.want)
		}
	}
}

func TestHandleIgnoresNonCreatedEvents(t *testing.T) {
	decisionWriter := &recordingTransactionRepository{}
	handler := riskHandler{
		riskRepo:       stubRiskRepository{},
		decisionWriter: decisionWriter,
		defaultRules:   defaultRuleConfigs(100000),
	}

	err := handler.Handle(context.Background(), &transactionevents.Envelope{
		EventID:   "event-1",
		EventType: transactionevents.EventTypeStatusChanged,
		After:     sampleRiskTransaction("100.00", time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if decisionWriter.called {
		t.Fatal("PublishDecision() should not be called for non-created events")
	}
}

func TestHandleRejectsWarningMatches(t *testing.T) {
	decisionWriter := &recordingTransactionRepository{}
	handler := riskHandler{
		riskRepo:       stubRiskRepository{},
		decisionWriter: decisionWriter,
		defaultRules:   defaultRuleConfigs(100000),
	}

	err := handler.Handle(context.Background(), newCreatedEnvelope(t, sampleRiskTransaction("150000.00", time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC))))
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if !decisionWriter.called {
		t.Fatal("PublishDecision() was not called")
	}
	if decisionWriter.approved {
		t.Fatal("PublishDecision() approved = true, want false")
	}
}

func TestHandleApprovesInfoMatches(t *testing.T) {
	decisionWriter := &recordingTransactionRepository{}
	cfgs := defaultRuleConfigs(100000)
	nightRule := cfgs[nightActivityRule]
	nightRule.Severity = "info"
	cfgs[nightActivityRule] = nightRule

	handler := riskHandler{
		riskRepo:       stubRiskRepository{},
		decisionWriter: decisionWriter,
		defaultRules:   cfgs,
	}

	err := handler.Handle(context.Background(), newCreatedEnvelope(t, sampleRiskTransaction("25000.00", time.Date(2026, 5, 15, 2, 15, 0, 0, time.UTC))))
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if !decisionWriter.called {
		t.Fatal("PublishDecision() was not called")
	}
	if !decisionWriter.approved {
		t.Fatal("PublishDecision() approved = false, want true")
	}
}

func sampleRiskTransaction(amount string, occurredAt time.Time) *transactionevents.TransactionPayload {
	return &transactionevents.TransactionPayload{
		ID:         "aaaaaaaa-1111-1111-1111-111111111111",
		UserID:     "11111111-1111-1111-1111-111111111111",
		Amount:     amount,
		Currency:   "RUB",
		Type:       "expense",
		Status:     "pending",
		OccurredAt: occurredAt,
		CreatedAt:  occurredAt,
		UpdatedAt:  occurredAt,
	}
}

type recordingTransactionRepository struct {
	called   bool
	txID     string
	approved bool
}

func (r *recordingTransactionRepository) PublishDecision(_ context.Context, tx *models.Transaction, approved bool) error {
	r.called = true
	if tx != nil {
		r.txID = tx.ID
	}
	r.approved = approved
	return nil
}

func newCreatedEnvelope(t *testing.T, tx *transactionevents.TransactionPayload) *transactionevents.Envelope {
	t.Helper()

	env, err := transactionevents.NewCreated(tx, tx.CreatedAt)
	if err != nil {
		t.Fatalf("NewCreated() error = %v", err)
	}
	return env
}
