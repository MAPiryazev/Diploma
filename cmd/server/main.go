package main

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
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/handlers/implementation"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/middleware"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/repository"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg, err := config.Load("../../environment/.env")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	defer func() {
		if database.Master != nil {
			database.Master.Close()
		}
		for _, slave := range database.Slaves {
			if slave != nil {
				slave.Close()
			}
		}
	}()

	if err := db.RunMigrations(database, "../../migrations"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	repos := &repository.Repositories{
		User:        repository.NewUserRepository(database),
		Account:     repository.NewAccountRepository(database),
		Category:    repository.NewCategoryRepository(database),
		Provider:    repository.NewProviderRepository(database),
		Transaction: repository.NewTransactionRepository(database),
		Analytics:   repository.NewAnalyticsRepository(database),
		Idempotency: repository.NewIdempotencyRepository(database),
	}

	services := service.NewServices(repos)
	handlers := implementation.NewHandlers(services, repos, database)

	router := http.NewServeMux()

	router.Handle("/", http.FileServer(http.Dir("../../web")))
	router.Handle("GET /metrics", promhttp.Handler())

	chain := middleware.NewChain().
		Use(middleware.RequestID).
		Use(middleware.Logger).
		Use(middleware.Metrics).
		Use(middleware.Recovery)

	router.Handle("GET /health", http.HandlerFunc(handlers.Health().Health))
	router.Handle("GET /ready", http.HandlerFunc(handlers.Health().Ready))

	txHandler := handlers.Transaction()
	router.Handle("POST /items", http.HandlerFunc(txHandler.CreateTransaction))
	router.Handle("GET /items", http.HandlerFunc(txHandler.ListTransactions))
	router.Handle("GET /items/{id}", http.HandlerFunc(txHandler.GetTransaction))
	router.Handle("PUT /items/{id}", http.HandlerFunc(txHandler.UpdateTransaction))
	router.Handle("DELETE /items/{id}", http.HandlerFunc(txHandler.DeleteTransaction))

	analyticsHandler := handlers.Analytics()
	router.Handle("GET /analytics", http.HandlerFunc(analyticsHandler.GetAnalytics))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      chain.Handle(router),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("failed to shutdown server: %v", err)
	}

	log.Println("server stopped")
}
