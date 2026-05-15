package main

import (
	"log"

	analyticsruntime "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/analytics/runtime"
)

func main() {
	if err := analyticsruntime.Run("../../environment/.env"); err != nil {
		log.Fatalf("analytics query service failed: %v", err)
	}
}
