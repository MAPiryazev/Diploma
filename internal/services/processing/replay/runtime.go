package replay

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/events"
	transactionevents "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/contracts/transactionevents"
	platformruntime "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/platform/runtime"
	"github.com/segmentio/kafka-go"
)

func Run(envPath string, args []string) error {
	flags := flag.NewFlagSet("dlqreplay", flag.ContinueOnError)
	limit := flags.Int("limit", 10, "max DLQ messages to read and replay (0 = no messages)")
	dryRun := flags.Bool("dry-run", false, "only print what would be replayed, do not write to main topic")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *limit < 0 {
		return errors.New("limit must be >= 0")
	}
	if *limit == 0 {
		log.Println("limit=0, nothing to do")
		return nil
	}

	cfg, err := platformruntime.LoadConfig(envPath)
	if err != nil {
		return err
	}
	if cfg.Kafka.Topic == "" || cfg.Kafka.DLQTopic == "" {
		return errors.New("kafka topic / dlq topic must be set")
	}
	if len(cfg.Kafka.Brokers) == 0 {
		return errors.New("kafka brokers empty")
	}

	replayWriter, err := events.NewReplayWriter(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	if err != nil {
		return fmt.Errorf("replay writer: %w", err)
	}
	defer func() { _ = replayWriter.Close() }()

	ctx := context.Background()
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Kafka.Brokers,
		GroupID:        "diploma-dlq-replay-tool",
		Topic:          cfg.Kafka.DLQTopic,
		MinBytes:       1,
		MaxBytes:       10e6,
		StartOffset:    kafka.FirstOffset,
		CommitInterval: 0,
	})
	defer func() { _ = r.Close() }()

	for i := 0; i < *limit; i++ {
		fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		msg, err := r.FetchMessage(fetchCtx)
		cancel()
		if err != nil {
			if err == context.DeadlineExceeded {
				log.Printf("no more DLQ messages (timeout after %d processed)", i)
				break
			}
			return fmt.Errorf("fetch: %w", err)
		}

		var wrap events.DLQMessage
		if err := json.Unmarshal(msg.Value, &wrap); err != nil {
			log.Printf("skip: invalid DLQ envelope at offset=%d: %v", msg.Offset, err)
			_ = r.CommitMessages(ctx, msg)
			continue
		}

		payload := []byte(wrap.Payload)
		if len(payload) == 0 {
			log.Printf("skip: empty payload offset=%d reason=%s", msg.Offset, wrap.Reason)
			_ = r.CommitMessages(ctx, msg)
			continue
		}

		if *dryRun {
			log.Printf("[dry-run] would replay offset=%d reason=%s event_id=%s bytes=%d", msg.Offset, wrap.Reason, wrap.EventID, len(payload))
			_ = r.CommitMessages(ctx, msg)
			continue
		}

		if err := republishPayload(ctx, replayWriter, payload); err != nil {
			log.Printf("replay failed offset=%d: %v (offset not committed)", msg.Offset, err)
			os.Exit(1)
		}

		log.Printf("replayed to topic %q offset=%d event_id=%s", cfg.Kafka.Topic, msg.Offset, wrap.EventID)
		if err := r.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
	}

	fmt.Println("dlqreplay done")
	return nil
}

func republishPayload(ctx context.Context, w *kafka.Writer, payload []byte) error {
	if len(payload) == 0 {
		return errors.New("payload is empty")
	}
	env, err := transactionevents.Parse(payload)
	if err != nil {
		return fmt.Errorf("payload is not a valid transaction event envelope: %w", err)
	}

	tx := transactionevents.TransactionForEvent(env)
	if tx == nil || tx.UserID == "" {
		return errors.New("transaction.user_id missing for routing key")
	}

	if err := w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(tx.UserID),
		Value: payload,
	}); err != nil {
		return fmt.Errorf("replay write: %w", err)
	}
	return nil
}
