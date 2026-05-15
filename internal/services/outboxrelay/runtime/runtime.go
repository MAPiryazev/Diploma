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

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/events"
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

	publisher, err := events.NewKafkaPublisher(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	if err != nil {
		return fmt.Errorf("init kafka publisher: %w", err)
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			log.Printf("kafka publisher close error: %v", err)
		}
	}()

	relay := events.NewOutboxRelay(
		events.NewPostgresOutboxStore(database),
		publisher,
		events.OutboxRelayConfig{},
	)

	healthHandler := platformhealth.New(database)
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.Handle("GET /health", http.HandlerFunc(healthHandler.Health))
	mux.Handle("GET /ready", http.HandlerFunc(healthHandler.Ready))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Relay.Port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("outbox relay service listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("relay server error: %v", err)
		}
	}()

	relayCtx, cancelRelay := context.WithCancel(context.Background())
	defer cancelRelay()
	go relay.Run(relayCtx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	cancelRelay()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown relay server: %w", err)
	}

	log.Println("outbox relay service stopped")
	return nil
}
