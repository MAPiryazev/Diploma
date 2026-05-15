package main

import (
	"log"

	projectionruntime "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/projectionbuilder/runtime"
)

func main() {
	if err := projectionruntime.Run("../../environment/.env"); err != nil {
		log.Fatalf("projection builder service failed: %v", err)
	}
}
