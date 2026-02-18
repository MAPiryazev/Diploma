package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	apperrors "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/errors"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/repository"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/validator"
)

type userServiceImpl struct {
	repo repository.UserRepository
}

func (s *userServiceImpl) CreateUser(ctx context.Context, req *CreateUserRequest) (*models.User, error) {
	if err := validator.ValidateUserName(req.Name); err != nil {
		return nil, err
	}
	if err := validator.ValidateEmail(req.Email); err != nil {
		return nil, err
	}

	user := &models.User{
		Name:  req.Name,
		Email: req.Email,
	}

	created, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return created, nil
}

func (s *userServiceImpl) GetUser(ctx context.Context, userID string) (*models.User, error) {
	if err := validator.ValidateUUID(userID); err != nil {
		return nil, err
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (s *userServiceImpl) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if err := validator.ValidateEmail(email); err != nil {
		return nil, err
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

type accountServiceImpl struct {
	repo repository.AccountRepository
}

func (s *accountServiceImpl) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*models.Account, error) {
	if err := validator.ValidateUUID(req.UserID); err != nil {
		return nil, err
	}
	if err := validator.ValidateAccountNumber(req.Number); err != nil {
		return nil, err
	}

	name := req.Name
	if len(name) == 0 || len(name) > 255 {
		return nil, &apperrors.ValidationError{Field: "name", Message: "length must be 1-255 characters"}
	}

	acc := &models.Account{
		UserID: req.UserID,
		Name:   name,
		Number: req.Number,
	}

	created, err := s.repo.Create(ctx, acc)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return created, nil
}

func (s *accountServiceImpl) GetAccount(ctx context.Context, accountID, userID string) (*models.Account, error) {
	if err := validator.ValidateUUID(accountID); err != nil {
		return nil, err
	}
	if err := validator.ValidateUUID(userID); err != nil {
		return nil, err
	}

	acc, err := s.repo.GetByID(ctx, accountID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return acc, nil
}

func (s *accountServiceImpl) ListAccounts(ctx context.Context, userID string) ([]*models.Account, error) {
	if err := validator.ValidateUUID(userID); err != nil {
		return nil, err
	}

	accs, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	if accs == nil {
		accs = make([]*models.Account, 0)
	}

	return accs, nil
}

type categoryServiceImpl struct {
	repo repository.CategoryRepository
}

func (s *categoryServiceImpl) CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*models.Category, error) {
	if err := validator.ValidateUUID(req.UserID); err != nil {
		return nil, err
	}
	if err := validator.ValidateCategoryName(req.Name); err != nil {
		return nil, err
	}

	cat := &models.Category{
		UserID: req.UserID,
		Name:   req.Name,
	}

	created, err := s.repo.Create(ctx, cat)
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return created, nil
}

func (s *categoryServiceImpl) GetCategory(ctx context.Context, categoryID, userID string) (*models.Category, error) {
	if err := validator.ValidateUUID(categoryID); err != nil {
		return nil, err
	}
	if err := validator.ValidateUUID(userID); err != nil {
		return nil, err
	}

	cat, err := s.repo.GetByID(ctx, categoryID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	return cat, nil
}

func (s *categoryServiceImpl) ListCategories(ctx context.Context, userID string) ([]*models.Category, error) {
	if err := validator.ValidateUUID(userID); err != nil {
		return nil, err
	}

	cats, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}

	if cats == nil {
		cats = make([]*models.Category, 0)
	}

	return cats, nil
}

type providerServiceImpl struct {
	repo repository.ProviderRepository
}

func (s *providerServiceImpl) CreateProvider(ctx context.Context, req *CreateProviderRequest) (*models.Provider, error) {
	if err := validator.ValidateProviderName(req.Name); err != nil {
		return nil, err
	}

	provider := &models.Provider{
		Name: req.Name,
	}

	created, err := s.repo.Create(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return created, nil
}

func (s *providerServiceImpl) GetProvider(ctx context.Context, providerID string) (*models.Provider, error) {
	if err := validator.ValidateUUID(providerID); err != nil {
		return nil, err
	}

	provider, err := s.repo.GetByID(ctx, providerID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return provider, nil
}

func (s *providerServiceImpl) GetProviderByName(ctx context.Context, name string) (*models.Provider, error) {
	if err := validator.ValidateProviderName(name); err != nil {
		return nil, err
	}

	provider, err := s.repo.GetByName(ctx, name)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return provider, nil
}

type transactionServiceImpl struct {
	txRepo  repository.TransactionRepository
	accRepo repository.AccountRepository
	catRepo repository.CategoryRepository
}

func (s *transactionServiceImpl) CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*models.Transaction, error) {
	if err := validator.ValidateUUID(req.UserID); err != nil {
		return nil, err
	}
	if err := validator.ValidateTransactionAmount(req.Amount); err != nil {
		return nil, err
	}
	if err := validator.ValidateTransactionType(req.Type); err != nil {
		return nil, err
	}
	if err := validator.ValidateTransactionStatus(req.Status); err != nil {
		return nil, err
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
		Status:        req.Status,
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

func (s *transactionServiceImpl) GetTransaction(ctx context.Context, txID, userID string) (*models.Transaction, error) {
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

func (s *transactionServiceImpl) ListTransactions(ctx context.Context, userID string) ([]*models.Transaction, error) {
	if err := validator.ValidateUUID(userID); err != nil {
		return nil, err
	}

	txs, err := s.txRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	if txs == nil {
		txs = make([]*models.Transaction, 0)
	}

	return txs, nil
}

func (s *transactionServiceImpl) UpdateTransaction(ctx context.Context, req *UpdateTransactionRequest) error {
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
	if err := validator.ValidateTransactionStatus(req.Status); err != nil {
		return err
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
		Status:        req.Status,
		Description:   req.Description,
		OccurredAt:    occurredAt,
	}

	err := s.txRepo.Update(ctx, tx)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return err
		}
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	return nil
}

func (s *transactionServiceImpl) DeleteTransaction(ctx context.Context, txID, userID string) error {
	if err := validator.ValidateUUID(txID); err != nil {
		return err
	}
	if err := validator.ValidateUUID(userID); err != nil {
		return err
	}

	err := s.txRepo.Delete(ctx, txID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return err
		}
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	return nil
}

type analyticsServiceImpl struct {
	repo repository.AnalyticsRepository
}

func (s *analyticsServiceImpl) GetAnalytics(ctx context.Context, userID, from, to string) (*AnalyticsResponse, error) {
	if err := validator.ValidateUUID(userID); err != nil {
		return nil, err
	}
	if err := validator.ValidateDateRange(from, to); err != nil {
		return nil, err
	}

	sum, err := s.repo.GetSum(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get sum: %w", err)
	}

	avg, err := s.repo.GetAvg(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get avg: %w", err)
	}

	count, err := s.repo.GetCount(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get count: %w", err)
	}

	median, err := s.repo.GetMedian(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get median: %w", err)
	}

	p90, err := s.repo.GetPercentile90(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get percentile 90: %w", err)
	}

	return &AnalyticsResponse{
		Sum:          sum,
		Avg:          avg,
		Count:        count,
		Median:       median,
		Percentile90: p90,
	}, nil
}

func (s *analyticsServiceImpl) GetSum(ctx context.Context, userID, from, to string) (string, error) {
	if err := validator.ValidateUUID(userID); err != nil {
		return "", err
	}
	if err := validator.ValidateDateRange(from, to); err != nil {
		return "", err
	}

	sum, err := s.repo.GetSum(ctx, userID, from, to)
	if err != nil {
		return "", fmt.Errorf("failed to get sum: %w", err)
	}

	return sum, nil
}

func (s *analyticsServiceImpl) GetAvg(ctx context.Context, userID, from, to string) (string, error) {
	if err := validator.ValidateUUID(userID); err != nil {
		return "", err
	}
	if err := validator.ValidateDateRange(from, to); err != nil {
		return "", err
	}

	avg, err := s.repo.GetAvg(ctx, userID, from, to)
	if err != nil {
		return "", fmt.Errorf("failed to get avg: %w", err)
	}

	return avg, nil
}

func (s *analyticsServiceImpl) GetCount(ctx context.Context, userID, from, to string) (int64, error) {
	if err := validator.ValidateUUID(userID); err != nil {
		return 0, err
	}
	if err := validator.ValidateDateRange(from, to); err != nil {
		return 0, err
	}

	count, err := s.repo.GetCount(ctx, userID, from, to)
	if err != nil {
		return 0, fmt.Errorf("failed to get count: %w", err)
	}

	return count, nil
}

func (s *analyticsServiceImpl) GetMedian(ctx context.Context, userID, from, to string) (string, error) {
	if err := validator.ValidateUUID(userID); err != nil {
		return "", err
	}
	if err := validator.ValidateDateRange(from, to); err != nil {
		return "", err
	}

	median, err := s.repo.GetMedian(ctx, userID, from, to)
	if err != nil {
		return "", fmt.Errorf("failed to get median: %w", err)
	}

	return median, nil
}

func (s *analyticsServiceImpl) GetPercentile90(ctx context.Context, userID, from, to string) (string, error) {
	if err := validator.ValidateUUID(userID); err != nil {
		return "", err
	}
	if err := validator.ValidateDateRange(from, to); err != nil {
		return "", err
	}

	p90, err := s.repo.GetPercentile90(ctx, userID, from, to)
	if err != nil {
		return "", fmt.Errorf("failed to get percentile 90: %w", err)
	}

	return p90, nil
}
