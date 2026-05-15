package config

import (
	"reflect"
	"testing"
)

func TestParseAuthTokens(t *testing.T) {
	raw := "token-1:user-1:admin, token-2:user-2, broken, :missing-user, token-3:user-3: "

	got := parseAuthTokens(raw)
	want := []AuthToken{
		{Token: "token-1", UserID: "user-1", Role: "admin"},
		{Token: "token-2", UserID: "user-2", Role: "operator"},
		{Token: "token-3", UserID: "user-3", Role: "operator"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseAuthTokens() = %#v, want %#v", got, want)
	}
}

func TestEnvStringSlice(t *testing.T) {
	const name = "TEST_KAFKA_BROKERS"
	fallback := []string{"localhost:9094"}

	t.Run("uses fallback for empty env", func(t *testing.T) {
		t.Setenv(name, "")

		got := envStringSlice(name, fallback)
		if !reflect.DeepEqual(got, fallback) {
			t.Fatalf("envStringSlice() = %#v, want %#v", got, fallback)
		}
	})

	t.Run("splits and trims values", func(t *testing.T) {
		t.Setenv(name, " broker-1:9092, , broker-2:9092 ")

		got := envStringSlice(name, fallback)
		want := []string{"broker-1:9092", "broker-2:9092"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("envStringSlice() = %#v, want %#v", got, want)
		}
	})
}

func TestApplyEnvOverridesConfig(t *testing.T) {
	cfg := defaultConfig()

	t.Setenv("LEDGER_DB_HOST", "ledger-db.internal")
	t.Setenv("LEDGER_DB_PORT", "6432")
	t.Setenv("LEDGER_DB_NAME", "ledger_service")
	t.Setenv("ANALYTICS_DB_HOST", "analytics-db.internal")
	t.Setenv("ANALYTICS_DB_PORT", "7432")
	t.Setenv("ANALYTICS_DB_NAME", "analytics_service")
	t.Setenv("KAFKA_BROKERS", "kafka-1:9092,kafka-2:9092")
	t.Setenv("ANALYTICS_PORT", "18082")
	t.Setenv("RELAY_PORT", "18083")
	t.Setenv("SECURITY_AUTH_TOKENS", "prod-token:user-1:auditor")
	t.Setenv("MONITORING_LARGE_AMOUNT_THRESHOLD", "250000.5")

	applyEnv(cfg)

	if cfg.LedgerDatabase.Host != "ledger-db.internal" {
		t.Fatalf("LedgerDatabase.Host = %q", cfg.LedgerDatabase.Host)
	}
	if cfg.LedgerDatabase.Port != 6432 {
		t.Fatalf("LedgerDatabase.Port = %d", cfg.LedgerDatabase.Port)
	}
	if cfg.LedgerDatabase.Name != "ledger_service" {
		t.Fatalf("LedgerDatabase.Name = %q", cfg.LedgerDatabase.Name)
	}
	if cfg.AnalyticsDatabase.Host != "analytics-db.internal" {
		t.Fatalf("AnalyticsDatabase.Host = %q", cfg.AnalyticsDatabase.Host)
	}
	if cfg.AnalyticsDatabase.Port != 7432 {
		t.Fatalf("AnalyticsDatabase.Port = %d", cfg.AnalyticsDatabase.Port)
	}
	if cfg.AnalyticsDatabase.Name != "analytics_service" {
		t.Fatalf("AnalyticsDatabase.Name = %q", cfg.AnalyticsDatabase.Name)
	}
	if want := []string{"kafka-1:9092", "kafka-2:9092"}; !reflect.DeepEqual(cfg.Kafka.Brokers, want) {
		t.Fatalf("Kafka.Brokers = %#v, want %#v", cfg.Kafka.Brokers, want)
	}
	if cfg.Analytics.Port != 18082 {
		t.Fatalf("Analytics.Port = %d", cfg.Analytics.Port)
	}
	if cfg.Relay.Port != 18083 {
		t.Fatalf("Relay.Port = %d", cfg.Relay.Port)
	}
	if want := []AuthToken{{Token: "prod-token", UserID: "user-1", Role: "auditor"}}; !reflect.DeepEqual(cfg.Security.AuthTokens, want) {
		t.Fatalf("Security.AuthTokens = %#v, want %#v", cfg.Security.AuthTokens, want)
	}
	if cfg.Monitoring.LargeAmountThreshold != 250000.5 {
		t.Fatalf("LargeAmountThreshold = %v", cfg.Monitoring.LargeAmountThreshold)
	}
}
