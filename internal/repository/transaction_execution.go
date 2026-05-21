package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	apperrors "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/errors"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
)

type balanceAdjustmentKind string

const (
	balanceAdjustmentDebit  balanceAdjustmentKind = "debit"
	balanceAdjustmentCredit balanceAdjustmentKind = "credit"
)

type balanceAdjustment struct {
	AccountID string
	UserID    string
	Amount    string
	Field     string
	Kind      balanceAdjustmentKind
}

func buildBalanceAdjustments(tx *models.Transaction, reverse bool) []balanceAdjustment {
	if tx == nil || !isExecutedStatus(tx.Status) {
		return nil
	}

	fromID := derefString(tx.FromAccountID)
	toID := derefString(tx.ToAccountID)
	if reverse {
		switch tx.Type {
		case "income":
			return []balanceAdjustment{{AccountID: toID, UserID: tx.UserID, Amount: tx.Amount, Field: "to_account_id", Kind: balanceAdjustmentDebit}}
		case "expense":
			return []balanceAdjustment{{AccountID: fromID, UserID: tx.UserID, Amount: tx.Amount, Field: "from_account_id", Kind: balanceAdjustmentCredit}}
		case "transfer":
			return []balanceAdjustment{
				{AccountID: toID, UserID: tx.UserID, Amount: tx.Amount, Field: "to_account_id", Kind: balanceAdjustmentDebit},
				{AccountID: fromID, UserID: tx.UserID, Amount: tx.Amount, Field: "from_account_id", Kind: balanceAdjustmentCredit},
			}
		}
		return nil
	}

	switch tx.Type {
	case "income":
		return []balanceAdjustment{{AccountID: toID, UserID: tx.UserID, Amount: tx.Amount, Field: "to_account_id", Kind: balanceAdjustmentCredit}}
	case "expense":
		return []balanceAdjustment{{AccountID: fromID, UserID: tx.UserID, Amount: tx.Amount, Field: "from_account_id", Kind: balanceAdjustmentDebit}}
	case "transfer":
		return []balanceAdjustment{
			{AccountID: fromID, UserID: tx.UserID, Amount: tx.Amount, Field: "from_account_id", Kind: balanceAdjustmentDebit},
			{AccountID: toID, UserID: tx.UserID, Amount: tx.Amount, Field: "to_account_id", Kind: balanceAdjustmentCredit},
		}
	}
	return nil
}

func (r *transactionRepository) reconcileExecutionBalances(
	ctx context.Context,
	dbTx *sql.Tx,
	before *models.Transaction,
	after *models.Transaction,
) error {
	for _, adjustment := range buildBalanceAdjustments(before, true) {
		if err := r.applyBalanceAdjustment(ctx, dbTx, adjustment); err != nil {
			return err
		}
	}
	for _, adjustment := range buildBalanceAdjustments(after, false) {
		if err := r.applyBalanceAdjustment(ctx, dbTx, adjustment); err != nil {
			return err
		}
	}
	return nil
}

func (r *transactionRepository) applyBalanceAdjustment(ctx context.Context, dbTx *sql.Tx, adj balanceAdjustment) error {
	switch adj.Kind {
	case balanceAdjustmentCredit:
		return creditAccountBalance(ctx, dbTx, adj)
	case balanceAdjustmentDebit:
		return debitAccountBalance(ctx, dbTx, adj)
	default:
		return fmt.Errorf("unsupported balance adjustment kind %q", adj.Kind)
	}
}

func creditAccountBalance(ctx context.Context, dbTx *sql.Tx, adj balanceAdjustment) error {
	if adj.AccountID == "" {
		return &apperrors.ValidationError{Field: adj.Field, Message: "account is required for executed transaction"}
	}

	const q = `
		UPDATE accounts
		SET balance = balance + $1::numeric
		WHERE id = $2 AND user_id = $3
	`
	res, err := dbTx.ExecContext(ctx, q, adj.Amount, adj.AccountID, adj.UserID)
	if err != nil {
		return fmt.Errorf("credit account balance: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("credit account balance rows: %w", err)
	}
	if rows == 0 {
		return &apperrors.ValidationError{Field: adj.Field, Message: "account not found"}
	}
	return nil
}

func debitAccountBalance(ctx context.Context, dbTx *sql.Tx, adj balanceAdjustment) error {
	if adj.AccountID == "" {
		return &apperrors.ValidationError{Field: adj.Field, Message: "account is required for executed transaction"}
	}

	const lockQuery = `
		SELECT balance >= $1::numeric
		FROM accounts
		WHERE id = $2 AND user_id = $3
		FOR UPDATE
	`

	var enoughFunds bool
	if err := dbTx.QueryRowContext(ctx, lockQuery, adj.Amount, adj.AccountID, adj.UserID).Scan(&enoughFunds); err != nil {
		if err == sql.ErrNoRows {
			return &apperrors.ValidationError{Field: adj.Field, Message: "account not found"}
		}
		return fmt.Errorf("lock account balance: %w", err)
	}
	if !enoughFunds {
		return &apperrors.ValidationError{Field: adj.Field, Message: "insufficient funds"}
	}

	const updateQuery = `
		UPDATE accounts
		SET balance = balance - $1::numeric
		WHERE id = $2 AND user_id = $3
	`
	if _, err := dbTx.ExecContext(ctx, updateQuery, adj.Amount, adj.AccountID, adj.UserID); err != nil {
		return fmt.Errorf("debit account balance: %w", err)
	}
	return nil
}

func isExecutedStatus(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "done")
}

func isPendingStatus(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "pending")
}

func isExecutionRejection(err error) bool {
	var validationErr *apperrors.ValidationError
	return errors.As(err, &validationErr)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func cloneTransaction(tx *models.Transaction) *models.Transaction {
	if tx == nil {
		return nil
	}

	clone := *tx
	return &clone
}
