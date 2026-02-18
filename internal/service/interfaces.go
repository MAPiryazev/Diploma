package service

import (
	"context"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
)

type UserService interface {
	CreateUser(ctx context.Context, req *CreateUserRequest) (*models.User, error)
	GetUser(ctx context.Context, userID string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
}

type AccountService interface {
	CreateAccount(ctx context.Context, req *CreateAccountRequest) (*models.Account, error)
	GetAccount(ctx context.Context, accountID, userID string) (*models.Account, error)
	ListAccounts(ctx context.Context, userID string) ([]*models.Account, error)
}

type CategoryService interface {
	CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*models.Category, error)
	GetCategory(ctx context.Context, categoryID, userID string) (*models.Category, error)
	ListCategories(ctx context.Context, userID string) ([]*models.Category, error)
}

type ProviderService interface {
	CreateProvider(ctx context.Context, req *CreateProviderRequest) (*models.Provider, error)
	GetProvider(ctx context.Context, providerID string) (*models.Provider, error)
	GetProviderByName(ctx context.Context, name string) (*models.Provider, error)
}

type TransactionService interface {
	CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*models.Transaction, error)
	GetTransaction(ctx context.Context, txID, userID string) (*models.Transaction, error)
	ListTransactions(ctx context.Context, userID string) ([]*models.Transaction, error)
	UpdateTransaction(ctx context.Context, req *UpdateTransactionRequest) error
	DeleteTransaction(ctx context.Context, txID, userID string) error
}

type AnalyticsService interface {
	GetAnalytics(ctx context.Context, userID, from, to string) (*AnalyticsResponse, error)
	GetSum(ctx context.Context, userID, from, to string) (string, error)
	GetAvg(ctx context.Context, userID, from, to string) (string, error)
	GetCount(ctx context.Context, userID, from, to string) (int64, error)
	GetMedian(ctx context.Context, userID, from, to string) (string, error)
	GetPercentile90(ctx context.Context, userID, from, to string) (string, error)
}

// DTO Requests
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CreateAccountRequest struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Number string `json:"number"`
}

type CreateCategoryRequest struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
}

type CreateProviderRequest struct {
	Name string `json:"name"`
}

type CreateTransactionRequest struct {
	UserID        string  `json:"user_id"`
	Amount        string  `json:"amount"`
	Currency      string  `json:"currency"`
	FromAccountID *string `json:"from_account_id"`
	ToAccountID   *string `json:"to_account_id"`
	ProviderID    *string `json:"provider_id"`
	CategoryID    *string `json:"category_id"`
	Type          string  `json:"type"`
	Status        string  `json:"status"`
	Description   *string `json:"description"`
	ExternalID    *string `json:"external_id"`
	OccurredAt    string  `json:"occurred_at"`
}

type UpdateTransactionRequest struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id"`
	Amount        string  `json:"amount"`
	Currency      string  `json:"currency"`
	FromAccountID *string `json:"from_account_id"`
	ToAccountID   *string `json:"to_account_id"`
	ProviderID    *string `json:"provider_id"`
	CategoryID    *string `json:"category_id"`
	Type          string  `json:"type"`
	Status        string  `json:"status"`
	Description   *string `json:"description"`
	OccurredAt    string  `json:"occurred_at"`
}

// DTO Responses
type AnalyticsResponse struct {
	Sum          string `json:"sum"`
	Avg          string `json:"avg"`
	Count        int64  `json:"count"`
	Median       string `json:"median"`
	Percentile90 string `json:"percentile_90"`
}
