package runtime

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/db"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/middleware"
	analyticsapp "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/analytics/app"
	analyticshttp "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/analytics/http"
	analyticspostgres "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/analytics/postgres"
	platformhealth "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/platform/httphealth"
	platformruntime "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/platform/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Run(envPath string) error {
	cfg, err := platformruntime.LoadConfig(envPath)
	if err != nil {
		return err
	}

	database, err := platformruntime.ConnectAnalyticsDatabase(cfg)
	if err != nil {
		return err
	}
	defer platformruntime.CloseDatabase(database)

	if err := db.RunMigrations(database, "../../migrations/analytics"); err != nil {
		return fmt.Errorf("run analytics migrations: %w", err)
	}

	svc := analyticsapp.NewService(analyticsRepositoryAdapter{repo: analyticspostgres.NewRepository(database)})
	handler := analyticshttp.NewHandler(svc)
	healthHandler := platformhealth.New(database)
	auth := middleware.JWTAuth(cfg.Security.JWTSecret, cfg.Security.AuthTokens)

	router := http.NewServeMux()
	router.Handle("GET /metrics", promhttp.Handler())
	router.Handle("GET /health", http.HandlerFunc(healthHandler.Health))
	router.Handle("GET /ready", http.HandlerFunc(healthHandler.Ready))
	router.Handle("GET /analytics", auth(http.HandlerFunc(handler.GetAnalytics)))
	router.Handle("GET /analytics/stream", auth(http.HandlerFunc(handler.GetStreamAnalytics)))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Analytics.Port),
		Handler:      middleware.NewChain().Use(middleware.CORS).Use(middleware.RequestID).Use(middleware.Logger).Use(middleware.Metrics).Use(middleware.Recovery).Handle(router),
		ReadTimeout:  platformruntime.DurationSeconds(cfg.Server.ReadTimeout, 10*time.Second),
		WriteTimeout: platformruntime.DurationSeconds(cfg.Server.WriteTimeout, 10*time.Second),
	}

	go func() {
		log.Printf("analytics query service listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("analytics server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown analytics server: %w", err)
	}

	log.Println("analytics query service stopped")
	return nil
}

type analyticsRepositoryAdapter struct {
	repo *analyticspostgres.Repository
}

func (a analyticsRepositoryAdapter) GetSum(ctx context.Context, userID string, from, to string) (string, error) {
	return a.repo.GetSum(ctx, userID, from, to)
}

func (a analyticsRepositoryAdapter) GetAvg(ctx context.Context, userID string, from, to string) (string, error) {
	return a.repo.GetAvg(ctx, userID, from, to)
}

func (a analyticsRepositoryAdapter) GetCount(ctx context.Context, userID string, from, to string) (int64, error) {
	return a.repo.GetCount(ctx, userID, from, to)
}

func (a analyticsRepositoryAdapter) GetMedian(ctx context.Context, userID string, from, to string) (string, error) {
	return a.repo.GetMedian(ctx, userID, from, to)
}

func (a analyticsRepositoryAdapter) GetPercentile90(ctx context.Context, userID string, from, to string) (string, error) {
	return a.repo.GetPercentile90(ctx, userID, from, to)
}

func (a analyticsRepositoryAdapter) GetStreamStats(ctx context.Context, userID string, from, to string) ([]analyticsapp.StreamAnalyticsRow, error) {
	rows, err := a.repo.GetStreamStats(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}

	out := make([]analyticsapp.StreamAnalyticsRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, analyticsapp.StreamAnalyticsRow{
			StatDate:           row.StatDate,
			Currency:           row.Currency,
			CreatedCount:       row.CreatedCount,
			UpdatedCount:       row.UpdatedCount,
			DeletedCount:       row.DeletedCount,
			StatusChangedCount: row.StatusChangedCount,
			CreatedAmount:      row.CreatedAmount,
			LastEventTime:      row.LastEventTime,
		})
	}
	return out, nil
}
