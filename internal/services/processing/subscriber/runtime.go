package subscriber

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/db"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/events"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/observability"
	processingstore "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/processing/store"
	transactionevents "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/contracts/transactionevents"
	platformhealth "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/platform/httphealth"
	platformruntime "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/platform/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	"github.com/wb-go/wbf/dbpg"
)

const (
	handlerMaxAttempts = 3
	handlerRetryBase   = 100 * time.Millisecond
)

type Handler interface {
	Handle(ctx context.Context, env *transactionevents.Envelope) error
}

type HandlerFactory func(database *dbpg.DB) Handler

type Options struct {
	ServiceName    string
	SubscriberName string
}

func Run(envPath string, options Options, newHandler HandlerFactory) error {
	cfg, err := platformruntime.LoadConfig(envPath)
	if err != nil {
		return err
	}
	if newHandler == nil {
		return errors.New("subscriber handler factory is nil")
	}
	if options.ServiceName == "" {
		return errors.New("service name is empty")
	}
	if options.SubscriberName == "" {
		return errors.New("subscriber name is empty")
	}
	if cfg.Kafka.ConsumerGroupID == "" {
		return errors.New("kafka consumer group id is empty")
	}
	if cfg.Kafka.Topic == "" {
		return errors.New("kafka topic is empty")
	}
	if cfg.Kafka.DLQTopic == "" {
		return errors.New("kafka dlq topic is empty")
	}
	if len(cfg.Kafka.Brokers) == 0 {
		return errors.New("kafka brokers are empty")
	}

	database, err := platformruntime.ConnectAnalyticsDatabase(cfg)
	if err != nil {
		return err
	}
	defer platformruntime.CloseDatabase(database)

	if err := db.RunMigrations(database, "../../migrations/analytics"); err != nil {
		return fmt.Errorf("run analytics migrations: %w", err)
	}

	processedStore := processingstore.NewPostgresProcessedEventsStore(database)
	handler := newHandler(database)
	if handler == nil {
		return errors.New("subscriber handler is nil")
	}
	dlqPublisher, err := events.NewDLQPublisher(cfg.Kafka.Brokers, cfg.Kafka.DLQTopic)
	if err != nil {
		return fmt.Errorf("init DLQ publisher: %w", err)
	}
	defer func() {
		if err := dlqPublisher.Close(); err != nil {
			log.Printf("%s dlq publisher close error: %v", options.ServiceName, err)
		}
	}()

	healthHandler := platformhealth.New(database)
	metricsAddr := fmt.Sprintf(":%d", cfg.Consumer.MetricsPort)
	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", promhttp.Handler())
	metricsMux.Handle("GET /health", http.HandlerFunc(healthHandler.Health))
	metricsMux.Handle("GET /ready", http.HandlerFunc(healthHandler.Ready))
	metricsServer := &http.Server{
		Addr:    metricsAddr,
		Handler: metricsMux,
	}

	go func() {
		log.Printf("%s metrics listening on %s", options.ServiceName, metricsAddr)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("%s metrics server error: %v", options.ServiceName, err)
		}
	}()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Kafka.Brokers,
		GroupID:        cfg.Kafka.ConsumerGroupID,
		Topic:          cfg.Kafka.Topic,
		MinBytes:       1,
		MaxBytes:       10e6,
		StartOffset:    kafka.FirstOffset,
		CommitInterval: 0,
	})
	defer func() {
		if err := reader.Close(); err != nil {
			log.Printf("%s kafka reader close error: %v", options.ServiceName, err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runConsumeLoop(ctx, reader, processedStore, dlqPublisher, options, handler)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("%s received signal %s, shutting down", options.ServiceName, sig)
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("%s consume loop exited: %v", options.ServiceName, err)
		}
	}

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("%s metrics server shutdown error: %v", options.ServiceName, err)
	}

	log.Printf("%s stopped", options.ServiceName)
	return nil
}

func runConsumeLoop(
	ctx context.Context,
	reader *kafka.Reader,
	processedStore processingstore.ProcessedEventsStore,
	dlqPublisher *events.DLQPublisher,
	options Options,
	handler Handler,
) error {
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			return fmt.Errorf("fetch message: %w", err)
		}
		handleMessage(ctx, reader, msg, processedStore, dlqPublisher, options, handler)
	}
}

func handleMessage(
	ctx context.Context,
	reader *kafka.Reader,
	msg kafka.Message,
	processedStore processingstore.ProcessedEventsStore,
	dlqPublisher *events.DLQPublisher,
	options Options,
	handler Handler,
) {
	env, err := transactionevents.Parse(msg.Value)
	if err != nil {
		observability.RecordKafkaConsumerInvalid()
		log.Printf("%s invalid message: partition=%d offset=%d err=%v", options.ServiceName, msg.Partition, msg.Offset, err)
		publishDLQ(ctx, dlqPublisher, msg, nil, "validation_failed", options.ServiceName)
		commitMessage(ctx, reader, msg, options.ServiceName)
		return
	}

	isNew, err := processedStore.SaveIfNew(ctx, options.SubscriberName, env.EventID, env.EventType)
	if err != nil {
		log.Printf("%s save processed event failed: subscriber=%s event_id=%s partition=%d offset=%d err=%v",
			options.ServiceName, options.SubscriberName, env.EventID, msg.Partition, msg.Offset, err)
		publishDLQ(ctx, dlqPublisher, msg, env, "save_processed_event_failed", options.ServiceName)
		commitMessage(ctx, reader, msg, options.ServiceName)
		return
	}
	if !isNew {
		observability.RecordKafkaConsumerDuplicate()
		log.Printf("%s duplicate event skipped: subscriber=%s event_id=%s partition=%d offset=%d",
			options.ServiceName, options.SubscriberName, env.EventID, msg.Partition, msg.Offset)
		commitMessage(ctx, reader, msg, options.ServiceName)
		return
	}

	start := time.Now()
	var lastErr error
	backoff := handlerRetryBase
	for attempt := 0; attempt < handlerMaxAttempts; attempt++ {
		if attempt > 0 {
			observability.RecordKafkaConsumerHandlerRetry()
			log.Printf("%s handler retry attempt=%d subscriber=%s event_id=%s partition=%d offset=%d",
				options.ServiceName, attempt+1, options.SubscriberName, env.EventID, msg.Partition, msg.Offset)
			select {
			case <-ctx.Done():
				log.Printf("%s handler retry cancelled: %v", options.ServiceName, ctx.Err())
				publishDLQ(ctx, dlqPublisher, msg, env, "handler_failed", options.ServiceName)
				commitMessage(ctx, reader, msg, options.ServiceName)
				return
			case <-time.After(backoff):
			}
			backoff *= 2
		}

		lastErr = handler.Handle(ctx, env)
		if lastErr == nil {
			observability.ObserveKafkaConsumerHandleDuration(time.Since(start))
			observability.ObserveKafkaConsumerEventProcessingLag(env.EventTime)
			observability.ObserveKafkaConsumerKafkaProcessingLag(msg.Time)
			observability.RecordKafkaConsumerProcessed(env.EventType)
			commitMessage(ctx, reader, msg, options.ServiceName)
			return
		}

		log.Printf("%s process event failed (attempt %d/%d): subscriber=%s event_id=%s partition=%d offset=%d err=%v",
			options.ServiceName, attempt+1, handlerMaxAttempts, options.SubscriberName, env.EventID, msg.Partition, msg.Offset, lastErr)
	}

	publishDLQ(ctx, dlqPublisher, msg, env, "handler_failed", options.ServiceName)
	commitMessage(ctx, reader, msg, options.ServiceName)
}

func commitMessage(ctx context.Context, reader *kafka.Reader, msg kafka.Message, serviceName string) {
	if err := reader.CommitMessages(ctx, msg); err != nil {
		observability.RecordKafkaConsumerCommitError()
		log.Printf("%s commit message failed: partition=%d offset=%d err=%v", serviceName, msg.Partition, msg.Offset, err)
	}
}

func publishDLQ(
	ctx context.Context,
	publisher *events.DLQPublisher,
	msg kafka.Message,
	env *transactionevents.Envelope,
	reason string,
	serviceName string,
) {
	eventID := ""
	eventType := ""
	if env != nil {
		eventID = env.EventID
		eventType = env.EventType
	}

	if err := publisher.Publish(ctx, events.DLQMessage{
		FailedAt:      time.Now().UTC(),
		OriginalTopic: msg.Topic,
		Partition:     msg.Partition,
		Offset:        msg.Offset,
		EventID:       eventID,
		EventType:     eventType,
		Reason:        reason,
		Payload:       string(msg.Value),
	}); err != nil {
		observability.RecordKafkaConsumerDLQError()
		log.Printf("%s publish dlq failed: partition=%d offset=%d reason=%s err=%v", serviceName, msg.Partition, msg.Offset, reason, err)
		return
	}
	observability.RecordKafkaConsumerDLQPublished()
}
