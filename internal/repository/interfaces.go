package repository

import (
	"context"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
)

type TransactionRepository interface {
	Create(ctx context.Context, tx *models.Transaction) (*models.Transaction, error)
	GetByID(ctx context.Context, id, userID string) (*models.Transaction, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*models.Transaction, error)
	Update(ctx context.Context, tx *models.Transaction) error
	Delete(ctx context.Context, id, userID string) error
	ResolvePending(ctx context.Context, id string, approved bool) (*models.Transaction, bool, error)
}

type TransactionDecisionPublisher interface {
	PublishDecision(ctx context.Context, tx *models.Transaction, approved bool) error
}

type AccountRepository interface {
	Create(ctx context.Context, acc *models.Account) (*models.Account, error)
	GetByID(ctx context.Context, id, userID string) (*models.Account, error)
	ListByUser(ctx context.Context, userID string) ([]*models.Account, error)
}

type CategoryRepository interface {
	Create(ctx context.Context, cat *models.Category) (*models.Category, error)
	GetByID(ctx context.Context, id, userID string) (*models.Category, error)
	ListByUser(ctx context.Context, userID string) ([]*models.Category, error)
}

type IdempotencyRecord struct {
	BodyHash     []byte
	HTTPStatus   int
	ResponseJSON []byte
}

type IdempotencyRepository interface {
	Get(ctx context.Context, userID, idempotencyKey string) (*IdempotencyRecord, error)
	Save(ctx context.Context, userID, idempotencyKey string, bodyHash []byte, httpStatus int, responseJSON []byte) error
}

type AuditRepository interface {
	Create(ctx context.Context, log *models.AuditLog) error
}
