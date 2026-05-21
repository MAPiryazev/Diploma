package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrors "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/errors"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/validator"
)

type TransactionRepository interface {
	Create(ctx context.Context, tx *models.Transaction) (*models.Transaction, error)
	GetByID(ctx context.Context, id, userID string) (*models.Transaction, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*models.Transaction, error)
	Update(ctx context.Context, tx *models.Transaction) error
	Delete(ctx context.Context, id, userID string) error
}

type AccountRepository interface {
	GetByID(ctx context.Context, id, userID string) (*models.Account, error)
}

type CategoryRepository interface {
	GetByID(ctx context.Context, id, userID string) (*models.Category, error)
}

type TransactionService interface {
	CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*models.Transaction, error)
	GetTransaction(ctx context.Context, txID, userID string) (*models.Transaction, error)
	ListTransactions(ctx context.Context, userID string, limit, offset int) ([]*models.Transaction, error)
	UpdateTransaction(ctx context.Context, req *UpdateTransactionRequest) error
	DeleteTransaction(ctx context.Context, txID, userID string) error
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
	ExternalID    *string `json:"external_id"`
	OccurredAt    string  `json:"occurred_at"`
}

type service struct {
	txRepo  TransactionRepository
	accRepo AccountRepository
	catRepo CategoryRepository
}

func NewService(txRepo TransactionRepository, accRepo AccountRepository, catRepo CategoryRepository) TransactionService {
	return &service{txRepo: txRepo, accRepo: accRepo, catRepo: catRepo}
}

func (s *service) CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*models.Transaction, error) {
	if err := validator.ValidateUUID(req.UserID); err != nil {
		return nil, err
	}
	if err := validator.ValidateTransactionAmount(req.Amount); err != nil {
		return nil, err
	}
	if err := validator.ValidateTransactionType(req.Type); err != nil {
		return nil, err
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		if err := validator.ValidateTransactionStatus(status); err != nil {
			return nil, err
		}
	}
	if err := validator.ValidateCurrency(req.Currency); err != nil {
		return nil, err
	}
	if err := validator.ValidateTimestamp(req.OccurredAt); err != nil {
		return nil, err
	}

	var fromID, toID string
	if req.FromAccountID != nil {
		fromID = *req.FromAccountID
	}
	if req.ToAccountID != nil {
		toID = *req.ToAccountID
	}

	if err := validator.ValidateTransactionAccounts(req.Type, fromID, toID); err != nil {
		return nil, err
	}
	if err := s.validateAccountOwnership(ctx, req.UserID, fromID, "from_account_id"); err != nil {
		return nil, err
	}
	if err := s.validateAccountOwnership(ctx, req.UserID, toID, "to_account_id"); err != nil {
		return nil, err
	}

	if req.CategoryID != nil {
		if err := validator.ValidateUUID(*req.CategoryID); err != nil {
			return nil, &apperrors.ValidationError{Field: "category_id", Message: "invalid UUID"}
		}
		_, err := s.catRepo.GetByID(ctx, *req.CategoryID, req.UserID)
		if err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				return nil, &apperrors.ValidationError{Field: "category_id", Message: "category not found"}
			}
			return nil, err
		}
	}

	occurredAt, _ := time.Parse(time.RFC3339, req.OccurredAt)
	tx := &models.Transaction{
		UserID:        req.UserID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		ProviderID:    req.ProviderID,
		CategoryID:    req.CategoryID,
		Type:          req.Type,
		Status:        "pending",
		Description:   req.Description,
		ExternalID:    req.ExternalID,
		OccurredAt:    occurredAt,
	}

	created, err := s.txRepo.Create(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}
	return created, nil
}

func (s *service) GetTransaction(ctx context.Context, txID, userID string) (*models.Transaction, error) {
	if err := validator.ValidateUUID(txID); err != nil {
		return nil, err
	}
	if err := validator.ValidateUUID(userID); err != nil {
		return nil, err
	}

	tx, err := s.txRepo.GetByID(ctx, txID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	return tx, nil
}

func (s *service) ListTransactions(ctx context.Context, userID string, limit, offset int) ([]*models.Transaction, error) {
	if err := validator.ValidateUUID(userID); err != nil {
		return nil, err
	}

	txs, err := s.txRepo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	if txs == nil {
		txs = make([]*models.Transaction, 0)
	}
	return txs, nil
}

func (s *service) UpdateTransaction(ctx context.Context, req *UpdateTransactionRequest) error {
	if err := validator.ValidateUUID(req.ID); err != nil {
		return err
	}
	if err := validator.ValidateUUID(req.UserID); err != nil {
		return err
	}
	if err := validator.ValidateTransactionAmount(req.Amount); err != nil {
		return err
	}
	if err := validator.ValidateTransactionType(req.Type); err != nil {
		return err
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		if err := validator.ValidateTransactionStatus(status); err != nil {
			return err
		}
	}
	if err := validator.ValidateCurrency(req.Currency); err != nil {
		return err
	}
	if err := validator.ValidateTimestamp(req.OccurredAt); err != nil {
		return err
	}

	var fromID, toID string
	if req.FromAccountID != nil {
		fromID = *req.FromAccountID
	}
	if req.ToAccountID != nil {
		toID = *req.ToAccountID
	}

	if err := validator.ValidateTransactionAccounts(req.Type, fromID, toID); err != nil {
		return err
	}
	if err := s.validateAccountOwnership(ctx, req.UserID, fromID, "from_account_id"); err != nil {
		return err
	}
	if err := s.validateAccountOwnership(ctx, req.UserID, toID, "to_account_id"); err != nil {
		return err
	}

	current, err := s.txRepo.GetByID(ctx, req.ID, req.UserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return err
		}
		return fmt.Errorf("failed to get transaction: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(current.Status), "pending") {
		return &apperrors.ValidationError{Field: "status", Message: "only pending transactions can be updated"}
	}
	if status := strings.TrimSpace(req.Status); status != "" && !strings.EqualFold(status, current.Status) {
		return &apperrors.ValidationError{Field: "status", Message: "status is managed asynchronously"}
	}

	occurredAt, _ := time.Parse(time.RFC3339, req.OccurredAt)
	tx := &models.Transaction{
		ID:            req.ID,
		UserID:        req.UserID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		ProviderID:    req.ProviderID,
		CategoryID:    req.CategoryID,
		Type:          req.Type,
		Status:        current.Status,
		Description:   req.Description,
		ExternalID:    req.ExternalID,
		OccurredAt:    occurredAt,
	}

	if err := s.txRepo.Update(ctx, tx); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return err
		}
		return fmt.Errorf("failed to update transaction: %w", err)
	}
	return nil
}

func (s *service) DeleteTransaction(ctx context.Context, txID, userID string) error {
	if err := validator.ValidateUUID(txID); err != nil {
		return err
	}
	if err := validator.ValidateUUID(userID); err != nil {
		return err
	}

	current, err := s.txRepo.GetByID(ctx, txID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return err
		}
		return fmt.Errorf("failed to get transaction: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(current.Status), "pending") {
		return &apperrors.ValidationError{Field: "status", Message: "only pending transactions can be deleted"}
	}

	if err := s.txRepo.Delete(ctx, txID, userID); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return err
		}
		return fmt.Errorf("failed to delete transaction: %w", err)
	}
	return nil
}

func (s *service) validateAccountOwnership(ctx context.Context, userID, accountID, field string) error {
	if accountID == "" {
		return nil
	}
	if err := validator.ValidateUUID(accountID); err != nil {
		return &apperrors.ValidationError{Field: field, Message: "invalid UUID"}
	}
	if _, err := s.accRepo.GetByID(ctx, accountID, userID); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return &apperrors.ValidationError{Field: field, Message: "account not found"}
		}
		return err
	}
	return nil
}
