// Утилита шага 6 (replay): читает сообщения из DLQ и заново публикует исходный payload в основной топик.
// Запуск (из корня репо): go run ./cmd/dlqreplay -limit 5
// Требует доступный Kafka (например localhost:9094 при docker compose).
//
// Если событие уже зафиксировано в processed_events для конкретного subscriber-а,
// projection-builder или risk-evaluation обработает повтор как дубликат.
// Чтобы реально переиграть бизнес-логику для того же event_id, перед replay удалите
// запись нужного subscriber-а из processed_events (ручной SQL / админка).
package main

import (
	"log"
	"os"

	replayruntime "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/processing/replay"
)

func main() {
	if err := replayruntime.Run("../../environment/.env", os.Args[1:]); err != nil {
		log.Fatalf("dlq replay failed: %v", err)
	}
}
