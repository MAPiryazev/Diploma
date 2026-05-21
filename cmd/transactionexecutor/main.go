package main

import (
	"log"

	executorruntime "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/transactionexecutor/runtime"
)

func main() {
	if err := executorruntime.Run("../../environment/.env"); err != nil {
		log.Fatalf("transaction executor service failed: %v", err)
	}
}
