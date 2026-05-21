package runtime

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	largeAmountRule      = "large_amount"
	velocity1hRule       = "velocity_1h"
	velocity24hAmountRule = "velocity_24h_amount"
	nightActivityRule    = "night_activity"
	roundAmountRule      = "round_amount"
	repeatedAmount24hRule = "repeated_amount_24h"
)

var ruleOrder = []string{
	largeAmountRule,
	velocity1hRule,
	velocity24hAmountRule,
	nightActivityRule,
	roundAmountRule,
	repeatedAmount24hRule,
}

type ruleParameters struct {
	ThresholdAmount      *float64 `json:"threshold_amount,omitempty"`
	WindowMinutes        *int     `json:"window_minutes,omitempty"`
	MaxTransactions      *int     `json:"max_transactions,omitempty"`
	MaxTotalAmount       *float64 `json:"max_total_amount,omitempty"`
	NightStartHour       *int     `json:"night_start_hour,omitempty"`
	NightEndHour         *int     `json:"night_end_hour,omitempty"`
	RoundModulo          *float64 `json:"round_modulo,omitempty"`
	MinAmount            *float64 `json:"min_amount,omitempty"`
	RepeatedTransactions *int     `json:"repeated_transactions,omitempty"`
}

type ruleConfig struct {
	RuleCode   string
	Enabled    bool
	Severity   string
	Parameters ruleParameters
	Version    int
	UpdatedAt  time.Time
}

type riskMatch struct {
	RuleCode string
	Severity string
	Reason   string
}

func defaultRuleConfigs(largeAmountThreshold float64) map[string]ruleConfig {
	if largeAmountThreshold <= 0 {
		largeAmountThreshold = 100000
	}

	return map[string]ruleConfig{
		largeAmountRule: {
			RuleCode: largeAmountRule,
			Enabled:  true,
			Severity: "warning",
			Version:  1,
			Parameters: ruleParameters{
				ThresholdAmount: float64Ptr(largeAmountThreshold),
			},
		},
		velocity1hRule: {
			RuleCode: velocity1hRule,
			Enabled:  true,
			Severity: "warning",
			Version:  1,
			Parameters: ruleParameters{
				WindowMinutes:   intPtr(60),
				MaxTransactions: intPtr(5),
			},
		},
		velocity24hAmountRule: {
			RuleCode: velocity24hAmountRule,
			Enabled:  true,
			Severity: "critical",
			Version:  1,
			Parameters: ruleParameters{
				WindowMinutes:  intPtr(24 * 60),
				MaxTotalAmount: float64Ptr(250000),
			},
		},
		nightActivityRule: {
			RuleCode: nightActivityRule,
			Enabled:  true,
			Severity: "info",
			Version:  1,
			Parameters: ruleParameters{
				NightStartHour: intPtr(0),
				NightEndHour:   intPtr(6),
				MinAmount:      float64Ptr(10000),
			},
		},
		roundAmountRule: {
			RuleCode: roundAmountRule,
			Enabled:  true,
			Severity: "info",
			Version:  1,
			Parameters: ruleParameters{
				RoundModulo: float64Ptr(1000),
				MinAmount:   float64Ptr(10000),
			},
		},
		repeatedAmount24hRule: {
			RuleCode: repeatedAmount24hRule,
			Enabled:  true,
			Severity: "warning",
			Version:  1,
			Parameters: ruleParameters{
				WindowMinutes:        intPtr(24 * 60),
				RepeatedTransactions: intPtr(3),
			},
		},
	}
}

func mergedRuleConfigs(defaults map[string]ruleConfig, overrides map[string]ruleConfig) []ruleConfig {
	configs := make([]ruleConfig, 0, len(ruleOrder))
	for _, ruleCode := range ruleOrder {
		cfg, ok := defaults[ruleCode]
		if !ok {
			continue
		}

		if override, ok := overrides[ruleCode]; ok {
			cfg.Enabled = override.Enabled
			if strings.TrimSpace(override.Severity) != "" {
				cfg.Severity = strings.TrimSpace(override.Severity)
			}
			if override.Version > 0 {
				cfg.Version = override.Version
			}
			if !override.UpdatedAt.IsZero() {
				cfg.UpdatedAt = override.UpdatedAt
			}
			cfg.Parameters = mergeRuleParameters(cfg.Parameters, override.Parameters)
		}

		if cfg.Enabled {
			configs = append(configs, cfg)
		}
	}
	return configs
}

func mergeRuleParameters(base, override ruleParameters) ruleParameters {
	out := base
	if override.ThresholdAmount != nil {
		out.ThresholdAmount = override.ThresholdAmount
	}
	if override.WindowMinutes != nil {
		out.WindowMinutes = override.WindowMinutes
	}
	if override.MaxTransactions != nil {
		out.MaxTransactions = override.MaxTransactions
	}
	if override.MaxTotalAmount != nil {
		out.MaxTotalAmount = override.MaxTotalAmount
	}
	if override.NightStartHour != nil {
		out.NightStartHour = override.NightStartHour
	}
	if override.NightEndHour != nil {
		out.NightEndHour = override.NightEndHour
	}
	if override.RoundModulo != nil {
		out.RoundModulo = override.RoundModulo
	}
	if override.MinAmount != nil {
		out.MinAmount = override.MinAmount
	}
	if override.RepeatedTransactions != nil {
		out.RepeatedTransactions = override.RepeatedTransactions
	}
	return out
}

func isNightHour(hour, start, end int) bool {
	if start == end {
		return true
	}
	if start < end {
		return hour >= start && hour < end
	}
	return hour >= start || hour < end
}

func isRoundAmount(amount, modulo float64) bool {
	if amount <= 0 || modulo <= 0 {
		return false
	}

	modulus := math.Mod(amount, modulo)
	return math.Abs(modulus) < 0.000001 || math.Abs(modulus-modulo) < 0.000001
}

func windowStart(at time.Time, windowMinutes int) time.Time {
	if windowMinutes <= 0 {
		windowMinutes = 1
	}
	return at.Add(-time.Duration(windowMinutes) * time.Minute)
}

func formatMoney(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}

func intPtr(value int) *int {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}

func intValue(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func float64Value(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}
