package main

import (
	"log"

	relayruntime "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/outboxrelay/runtime"
)

func main() {
	if err := relayruntime.Run("../../environment/.env"); err != nil {
		log.Fatalf("outbox relay service failed: %v", err)
	}
}
