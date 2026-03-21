// Утилита шага 6 (replay): читает сообщения из DLQ и заново публикует исходный payload в основной топик.
// Запуск (из корня репо): go run ./cmd/dlqreplay -limit 5
// Требует доступный Kafka (например localhost:9094 при docker compose).
//
// Если событие уже зафиксировано в processed_events, consumer обработает повтор как дубликат.
// Чтобы реально переиграть бизнес-логику для того же event_id, перед replay удалите строку
// из processed_events (ручной SQL / админка) — для диплома достаточно описать этот сценарий.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/config"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/events"
	"github.com/segmentio/kafka-go"
)

func main() {
	limit := flag.Int("limit", 10, "max DLQ messages to read and replay (0 = no messages)")
	dryRun := flag.Bool("dry-run", false, "only print what would be replayed, do not write to main topic")
	flag.Parse()

	if *limit < 0 {
		log.Fatal("limit must be >= 0")
	}
	if *limit == 0 {
		log.Println("limit=0, nothing to do")
		return
	}

	cfg, err := config.Load("../../environment/.env")
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.Kafka.Topic == "" || cfg.Kafka.DLQTopic == "" {
		log.Fatal("kafka topic / dlq topic must be set")
	}
	if len(cfg.Kafka.Brokers) == 0 {
		log.Fatal("kafka brokers empty")
	}

	replayWriter, err := events.NewReplayWriter(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	if err != nil {
		log.Fatalf("replay writer: %v", err)
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
			log.Fatalf("fetch: %v", err)
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

		if err := events.RepublishTransactionCreatedPayload(ctx, replayWriter, payload); err != nil {
			log.Printf("replay failed offset=%d: %v (offset not committed)", msg.Offset, err)
			os.Exit(1)
		}

		log.Printf("replayed to topic %q offset=%d event_id=%s", cfg.Kafka.Topic, msg.Offset, wrap.EventID)
		if err := r.CommitMessages(ctx, msg); err != nil {
			log.Fatalf("commit: %v", err)
		}
	}

	fmt.Println("dlqreplay done")
}
