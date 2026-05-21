package app

import (
	"context"
	"errors"
	"testing"
	"time"

	apperrors "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/errors"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
)

type stubTransactionRepo struct {
	createInput *models.Transaction
	getResult   *models.Transaction
	deleteID    string
	deleteUser  string
}

func (s *stubTransactionRepo) Create(_ context.Context, tx *models.Transaction) (*models.Transaction, error) {
	s.createInput = cloneModel(tx)
	out := cloneModel(tx)
	out.ID = "tx-1"
	out.CreatedAt = time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	out.UpdatedAt = out.CreatedAt
	return out, nil
}

func (s *stubTransactionRepo) GetByID(context.Context, string, string) (*models.Transaction, error) {
	if s.getResult == nil {
		return nil, apperrors.ErrNotFound
	}
	return cloneModel(s.getResult), nil
}

func (s *stubTransactionRepo) ListByUser(context.Context, string, int, int) ([]*models.Transaction, error) {
	return nil, nil
}

func (s *stubTransactionRepo) Update(context.Context, *models.Transaction) error {
	return nil
}

func (s *stubTransactionRepo) Delete(_ context.Context, id, userID string) error {
	s.deleteID = id
	s.deleteUser = userID
	return nil
}

type stubAccountRepo struct{}

func (stubAccountRepo) GetByID(context.Context, string, string) (*models.Account, error) {
	return &models.Account{}, nil
}

type stubCategoryRepo struct{}

func (stubCategoryRepo) GetByID(context.Context, string, string) (*models.Category, error) {
	return &models.Category{}, nil
}

func TestCreateTransactionForcesPendingStatus(t *testing.T) {
	txRepo := &stubTransactionRepo{}
	svc := NewService(txRepo, stubAccountRepo{}, stubCategoryRepo{})
	fromID := "11111111-1111-1111-1111-111111111112"

	created, err := svc.CreateTransaction(context.Background(), &CreateTransactionRequest{
		UserID:        "11111111-1111-1111-1111-111111111111",
		Amount:        "100.00",
		Currency:      "RUB",
		FromAccountID: &fromID,
		Type:          "expense",
		Status:        "done",
		OccurredAt:    time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("CreateTransaction() error = %v", err)
	}
	if txRepo.createInput == nil {
		t.Fatal("repository Create() was not called")
	}
	if txRepo.createInput.Status != "pending" {
		t.Fatalf("repository Create() status = %q, want pending", txRepo.createInput.Status)
	}
	if created.Status != "pending" {
		t.Fatalf("created status = %q, want pending", created.Status)
	}
}

func TestUpdateTransactionRejectsResolvedTransactions(t *testing.T) {
	txRepo := &stubTransactionRepo{
		getResult: &models.Transaction{
			ID:     "tx-1",
			UserID: "11111111-1111-1111-1111-111111111111",
			Status: "done",
		},
	}
	svc := NewService(txRepo, stubAccountRepo{}, stubCategoryRepo{})
	fromID := "11111111-1111-1111-1111-111111111112"

	err := svc.UpdateTransaction(context.Background(), &UpdateTransactionRequest{
		ID:            "tx-1",
		UserID:        "11111111-1111-1111-1111-111111111111",
		Amount:        "100.00",
		Currency:      "RUB",
		FromAccountID: &fromID,
		Type:          "expense",
		Status:        "done",
		OccurredAt:    time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
	})
	if err == nil {
		t.Fatal("UpdateTransaction() error = nil, want validation error")
	}
	var valErr *apperrors.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("UpdateTransaction() error = %T, want ValidationError", err)
	}
}

func TestDeleteTransactionRejectsResolvedTransactions(t *testing.T) {
	txRepo := &stubTransactionRepo{
		getResult: &models.Transaction{
			ID:     "tx-1",
			UserID: "11111111-1111-1111-1111-111111111111",
			Status: "failed",
		},
	}
	svc := NewService(txRepo, stubAccountRepo{}, stubCategoryRepo{})

	err := svc.DeleteTransaction(context.Background(), "tx-1", "11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("DeleteTransaction() error = nil, want validation error")
	}
	if txRepo.deleteID != "" {
		t.Fatal("repository Delete() should not be called")
	}
}

func cloneModel(tx *models.Transaction) *models.Transaction {
	if tx == nil {
		return nil
	}
	clone := *tx
	return &clone
}
