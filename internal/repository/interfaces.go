package repository

import (
	"context"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
)

type TransactionRepository interface {
	Create(ctx context.Context, tx *models.Transaction) (*models.Transaction, error)
	GetByID(ctx context.Context, id, userID string) (*models.Transaction, error)
	ListByUser(ctx context.Context, userID string) ([]*models.Transaction, error)
	Update(ctx context.Context, tx *models.Transaction) error
	Delete(ctx context.Context, id, userID string) error
}

type UserRepository interface {
	Create(ctx context.Context, user *models.User) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
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

type ProviderRepository interface {
	Create(ctx context.Context, provider *models.Provider) (*models.Provider, error)
	GetByID(ctx context.Context, id string) (*models.Provider, error)
	GetByName(ctx context.Context, name string) (*models.Provider, error)
}

type AnalyticsRepository interface {
	GetSum(ctx context.Context, userID string, from, to string) (string, error)
	GetAvg(ctx context.Context, userID string, from, to string) (string, error)
	GetCount(ctx context.Context, userID string, from, to string) (int64, error)
	GetMedian(ctx context.Context, userID string, from, to string) (string, error)
	GetPercentile90(ctx context.Context, userID string, from, to string) (string, error)
}

// Структура для инъекции зависимостей в сервисный слой
type Repositories struct {
	User        UserRepository
	Account     AccountRepository
	Category    CategoryRepository
	Provider    ProviderRepository
	Transaction TransactionRepository
	Analytics   AnalyticsRepository
}
