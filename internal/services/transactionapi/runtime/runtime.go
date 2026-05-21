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

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/config"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/db"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/middleware"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/repository"
	transactionapp "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/transactionapi/app"
	transactionhttp "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/transactionapi/http"
	transactionintegration "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/transactionapi/integrationevents"
	platformhealth "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/platform/httphealth"
	platformruntime "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/platform/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Run(envPath string) error {
	cfg, err := platformruntime.LoadConfig(envPath)
	if err != nil {
		return err
	}

	database, err := platformruntime.ConnectLedgerDatabase(cfg)
	if err != nil {
		return err
	}
	defer platformruntime.CloseDatabase(database)

	if err := db.RunMigrations(database, "../../migrations/ledger"); err != nil {
		return fmt.Errorf("run ledger migrations: %w", err)
	}

	txRepo := repository.NewTransactionRepository(database, transactionintegration.NewBuilder(), cfg.Kafka.Topic)
	accRepo := repository.NewAccountRepository(database)
	catRepo := repository.NewCategoryRepository(database)
	idemRepo := repository.NewIdempotencyRepository(database)
	auditRepo := repository.NewAuditRepository(database)

	svc := transactionapp.NewService(txRepo, accRepo, catRepo)
	handler := transactionhttp.NewHandler(svc, idempotencyAdapter{repo: idemRepo}, auditRepo)
	healthHandler := platformhealth.New(database)

	router := http.NewServeMux()
	router.Handle("/", http.FileServer(http.Dir("../../web")))
	router.Handle("GET /metrics", promhttp.Handler())
	router.Handle("GET /health", http.HandlerFunc(healthHandler.Health))
	router.Handle("GET /ready", http.HandlerFunc(healthHandler.Ready))

	auth := middleware.JWTAuth(cfg.Security.JWTSecret, cfg.Security.AuthTokens)
	router.Handle("POST /items", auth(http.HandlerFunc(handler.CreateTransaction)))
	router.Handle("GET /items", auth(http.HandlerFunc(handler.ListTransactions)))
	router.Handle("GET /items/{id}", auth(http.HandlerFunc(handler.GetTransaction)))
	router.Handle("PUT /items/{id}", auth(http.HandlerFunc(handler.UpdateTransaction)))
	router.Handle("DELETE /items/{id}", auth(http.HandlerFunc(handler.DeleteTransaction)))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      middleware.NewChain().Use(middleware.RequestID).Use(middleware.Logger).Use(middleware.Metrics).Use(middleware.Recovery).Handle(router),
		ReadTimeout:  platformruntime.DurationSeconds(cfg.Server.ReadTimeout, 10*time.Second),
		WriteTimeout: platformruntime.DurationSeconds(cfg.Server.WriteTimeout, 10*time.Second),
	}

	go func() {
		log.Printf("transaction api service listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("transaction api server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown transaction api: %w", err)
	}

	log.Println("transaction api service stopped")
	return nil
}

type idempotencyAdapter struct {
	repo repository.IdempotencyRepository
}

func (a idempotencyAdapter) Get(ctx context.Context, userID, idempotencyKey string) (*transactionhttp.IdempotencyRecord, error) {
	rec, err := a.repo.Get(ctx, userID, idempotencyKey)
	if err != nil || rec == nil {
		return nil, err
	}
	return &transactionhttp.IdempotencyRecord{
		BodyHash:     rec.BodyHash,
		HTTPStatus:   rec.HTTPStatus,
		ResponseJSON: rec.ResponseJSON,
	}, nil
}

func (a idempotencyAdapter) Save(ctx context.Context, userID, idempotencyKey string, bodyHash []byte, httpStatus int, responseJSON []byte) error {
	return a.repo.Save(ctx, userID, idempotencyKey, bodyHash, httpStatus, responseJSON)
}

func BuildConfig(envPath string) (*config.Config, error) {
	return platformruntime.LoadConfig(envPath)
}
