package main

import (
	"log"

	riskruntime "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/riskevaluation/runtime"
)

func main() {
	if err := riskruntime.Run("../../environment/.env"); err != nil {
		log.Fatalf("risk evaluation service failed: %v", err)
	}
}
