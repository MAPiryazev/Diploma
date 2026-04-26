package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/config"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/db"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/events"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/observability"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/security"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	"github.com/wb-go/wbf/dbpg"
)

const (
	handlerMaxAttempts = 3
	handlerRetryBase   = 100 * time.Millisecond
	largeAmountRule    = "large_amount"
)

func main() {
	cfg, err := config.Load("../../environment/.env")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if cfg.Kafka.ConsumerGroupID == "" {
		log.Fatalf("kafka consumer group id is empty")
	}
	if cfg.Kafka.Topic == "" {
		log.Fatalf("kafka topic is empty")
	}
	if cfg.Kafka.DLQTopic == "" {
		log.Fatalf("kafka dlq topic is empty")
	}
	if len(cfg.Kafka.Brokers) == 0 {
		log.Fatalf("kafka brokers are empty")
	}

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer closeDatabase(database)

	// Миграции выполняет только API (cmd/server), иначе гонка с consumer → duplicate pg_type.

	processedStore := events.NewPostgresProcessedEventsStore(database)
	monitoringStore := events.NewPostgresMonitoringEventsStore(database)
	statsStore := events.NewPostgresTransactionEventStatsStore(database)

	dlqPublisher, err := events.NewDLQPublisher(cfg.Kafka.Brokers, cfg.Kafka.DLQTopic)
	if err != nil {
		log.Fatalf("failed to init DLQ publisher: %v", err)
	}
	defer func() {
		if err := dlqPublisher.Close(); err != nil {
			log.Printf("dlq publisher close error: %v", err)
		}
	}()

	metricsAddr := fmt.Sprintf(":%d", cfg.Consumer.MetricsPort)
	metricsServer := &http.Server{
		Addr:    metricsAddr,
		Handler: promhttp.Handler(),
	}

	go func() {
		log.Printf("consumer metrics listening on %s", metricsAddr)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("metrics server error: %v", err)
		}
	}()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Kafka.Brokers,
		GroupID:        cfg.Kafka.ConsumerGroupID,
		Topic:          cfg.Kafka.Topic,
		MinBytes:       1,
		MaxBytes:       10e6,
		StartOffset:    kafka.FirstOffset,
		CommitInterval: 0, // manual commit after we decide message outcome
	})
	defer func() {
		if err := reader.Close(); err != nil {
			log.Printf("kafka reader close error: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runConsumeLoop(ctx, reader, processedStore, monitoringStore, statsStore, dlqPublisher, cfg.Monitoring.LargeAmountThreshold)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("received signal %s, shutting down", sig)
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("consume loop exited: %v", err)
		}
	}

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("metrics server shutdown error: %v", err)
	}

	log.Println("consumer stopped")
}

func closeDatabase(database *dbpg.DB) {
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

func runConsumeLoop(
	ctx context.Context,
	reader *kafka.Reader,
	processedStore events.ProcessedEventsStore,
	monitoringStore events.MonitoringEventsStore,
	statsStore events.TransactionEventStatsStore,
	dlqPublisher *events.DLQPublisher,
	largeAmountThreshold float64,
) error {
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			return fmt.Errorf("fetch message: %w", err)
		}

		handleMessage(ctx, reader, msg, processedStore, monitoringStore, statsStore, dlqPublisher, largeAmountThreshold)
	}
}

func handleMessage(
	ctx context.Context,
	reader *kafka.Reader,
	msg kafka.Message,
	processedStore events.ProcessedEventsStore,
	monitoringStore events.MonitoringEventsStore,
	statsStore events.TransactionEventStatsStore,
	dlqPublisher *events.DLQPublisher,
	largeAmountThreshold float64,
) {
	env, err := events.ParseTransactionEventJSON(msg.Value)
	if err != nil {
		observability.RecordKafkaConsumerInvalid()
		log.Printf("invalid message: partition=%d offset=%d err=%v", msg.Partition, msg.Offset, err)

		publishDLQ(ctx, dlqPublisher, msg, nil, "validation_failed")
		commitMessage(ctx, reader, msg)
		return
	}

	isNew, err := processedStore.SaveIfNew(ctx, env.EventID, env.EventType)
	if err != nil {
		log.Printf("save processed event failed: event_id=%s partition=%d offset=%d err=%v", security.MaskID(env.EventID), msg.Partition, msg.Offset, err)
		publishDLQ(ctx, dlqPublisher, msg, env, "save_processed_event_failed")
		commitMessage(ctx, reader, msg)
		return
	}

	if !isNew {
		observability.RecordKafkaConsumerDuplicate()
		log.Printf("duplicate event skipped: event_id=%s partition=%d offset=%d", security.MaskID(env.EventID), msg.Partition, msg.Offset)
		commitMessage(ctx, reader, msg)
		return
	}

	start := time.Now()
	var lastErr error
	backoff := handlerRetryBase
	for attempt := 0; attempt < handlerMaxAttempts; attempt++ {
		if attempt > 0 {
			observability.RecordKafkaConsumerHandlerRetry()
			log.Printf("handler retry attempt=%d event_id=%s partition=%d offset=%d", attempt+1, security.MaskID(env.EventID), msg.Partition, msg.Offset)
			select {
			case <-ctx.Done():
				log.Printf("handler retry cancelled: %v", ctx.Err())
				publishDLQ(ctx, dlqPublisher, msg, env, "handler_failed")
				commitMessage(ctx, reader, msg)
				return
			case <-time.After(backoff):
			}
			backoff *= 2
		}

		lastErr = processTransactionEvent(ctx, env, monitoringStore, statsStore, largeAmountThreshold)
		if lastErr == nil {
			observability.ObserveKafkaConsumerHandleDuration(time.Since(start))
			observability.ObserveKafkaConsumerEventProcessingLag(env.EventTime)
			observability.ObserveKafkaConsumerKafkaProcessingLag(msg.Time)
			observability.RecordKafkaConsumerProcessed(env.EventType)
			commitMessage(ctx, reader, msg)
			return
		}
		log.Printf("process event failed (attempt %d/%d): event_id=%s partition=%d offset=%d err=%v",
			attempt+1, handlerMaxAttempts, security.MaskID(env.EventID), msg.Partition, msg.Offset, lastErr)
	}

	publishDLQ(ctx, dlqPublisher, msg, env, "handler_failed")
	commitMessage(ctx, reader, msg)
}

func commitMessage(ctx context.Context, reader *kafka.Reader, msg kafka.Message) {
	if err := reader.CommitMessages(ctx, msg); err != nil {
		observability.RecordKafkaConsumerCommitError()
		log.Printf("commit message failed: partition=%d offset=%d err=%v", msg.Partition, msg.Offset, err)
	}
}

func publishDLQ(
	ctx context.Context,
	publisher *events.DLQPublisher,
	msg kafka.Message,
	env *events.TransactionEventEnvelope,
	reason string,
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
		log.Printf("publish dlq failed: partition=%d offset=%d reason=%s err=%v", msg.Partition, msg.Offset, reason, err)
		return
	}

	observability.RecordKafkaConsumerDLQPublished()
}

func processTransactionEvent(
	ctx context.Context,
	env *events.TransactionEventEnvelope,
	monitoringStore events.MonitoringEventsStore,
	statsStore events.TransactionEventStatsStore,
	largeAmountThreshold float64,
) error {
	tx := transactionForEvent(env)
	if tx == nil {
		return errors.New("transaction payload is nil")
	}

	log.Printf(
		"%s processed event_id=%s tx_id=%s user_id=%s amount=%s currency=%s type=%s status=%s",
		env.EventType,
		security.MaskID(env.EventID),
		security.MaskID(env.AggregateID),
		security.MaskID(tx.UserID),
		security.MaskAmount(tx.Amount),
		tx.Currency,
		tx.Type,
		tx.Status,
	)

	if statsStore != nil {
		if err := statsStore.Apply(ctx, buildTransactionEventStat(env, tx)); err != nil {
			return err
		}
		observability.RecordTransactionProjectionApplied(env.EventType)
	}

	if largeAmountThreshold > 0 && env.EventType != events.EventTypeTransactionDeleted {
		amount, err := strconv.ParseFloat(strings.TrimSpace(tx.Amount), 64)
		if err != nil {
			return fmt.Errorf("parse transaction amount: %w", err)
		}
		if amount >= largeAmountThreshold {
			observability.RecordMonitoringLargeAmountEvent()
			log.Printf(
				"monitoring rule matched rule=%s event_id=%s tx_id=%s user_id=%s threshold=%.2f",
				largeAmountRule,
				security.MaskID(env.EventID),
				security.MaskID(tx.ID),
				security.MaskID(tx.UserID),
				largeAmountThreshold,
			)
			if monitoringStore != nil {
				eventTime := env.EventTime
				if eventTime.IsZero() {
					eventTime = tx.OccurredAt
				}
				if err := monitoringStore.Save(ctx, events.MonitoringEvent{
					TransactionID: tx.ID,
					UserID:        tx.UserID,
					RuleCode:      largeAmountRule,
					Severity:      "warning",
					Reason:        "transaction amount exceeded configured threshold",
					EventTime:     eventTime,
				}); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func transactionForEvent(env *events.TransactionEventEnvelope) *models.Transaction {
	if env == nil {
		return nil
	}
	if env.After != nil {
		return env.After
	}
	if env.Transaction != nil {
		return env.Transaction
	}
	return env.Before
}

func buildTransactionEventStat(env *events.TransactionEventEnvelope, tx *models.Transaction) events.TransactionEventStat {
	stat := events.TransactionEventStat{
		UserID:    tx.UserID,
		Currency:  tx.Currency,
		StatDate:  tx.OccurredAt,
		EventTime: env.EventTime,
	}

	switch env.EventType {
	case events.EventTypeTransactionCreated:
		stat.CreatedCount = 1
		stat.CreatedAmount = tx.Amount
	case events.EventTypeTransactionUpdated:
		stat.UpdatedCount = 1
	case events.EventTypeTransactionDeleted:
		stat.DeletedCount = 1
	case events.EventTypeStatusChanged:
		stat.StatusChangedCount = 1
	}

	return stat
}
