package events

import (
	"context"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
)

type Publisher interface {
	PublishTransactionCreated(ctx context.Context, tx *models.Transaction) error
	Close() error
}
