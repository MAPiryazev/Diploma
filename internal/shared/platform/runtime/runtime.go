package runtime

import (
	"fmt"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/config"
	dbpkg "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/db"
	"github.com/wb-go/wbf/dbpg"
)

// LoadConfig is the shared platform entrypoint for process configuration.
func LoadConfig(envPath string) (*config.Config, error) {
	cfg, err := config.Load(envPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}

// ConnectDatabase keeps backward-compatible wiring to the ledger/write database.
func ConnectDatabase(cfg *config.Config) (*dbpg.DB, error) {
	database, err := dbpkg.Connect(cfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	return database, nil
}

// ConnectLedgerDatabase centralizes ledger/write DB wiring.
func ConnectLedgerDatabase(cfg *config.Config) (*dbpg.DB, error) {
	database, err := dbpkg.ConnectWithDatabase(cfg.LedgerDatabase)
	if err != nil {
		return nil, fmt.Errorf("connect ledger database: %w", err)
	}
	return database, nil
}

// ConnectAnalyticsDatabase centralizes analytics/read DB wiring.
func ConnectAnalyticsDatabase(cfg *config.Config) (*dbpg.DB, error) {
	database, err := dbpkg.ConnectWithDatabase(cfg.AnalyticsDatabase)
	if err != nil {
		return nil, fmt.Errorf("connect analytics database: %w", err)
	}
	return database, nil
}

// CloseDatabase gracefully shuts down master and read replicas if present.
func CloseDatabase(database *dbpg.DB) {
	if database == nil {
		return
	}
	if database.Master != nil {
		database.Master.Close()
	}
	for _, slave := range database.Slaves {
		if slave != nil {
			slave.Close()
		}
	}
}

func DurationSeconds(seconds int, fallback time.Duration) time.Duration {
	if seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
