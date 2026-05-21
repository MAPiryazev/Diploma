package main

import (
	"log"

	transactionruntime "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/transactionapi/runtime"
)

func main() {
	if err := transactionruntime.Run("../../environment/.env"); err != nil {
		log.Fatalf("transaction api service failed: %v", err)
	}
}
