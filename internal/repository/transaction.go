package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrors "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/errors"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/events"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	"github.com/wb-go/wbf/dbpg"
)

type transactionRepository struct {
	db    *dbpg.DB
	topic string
}

type transactionScanner interface {
	Scan(dest ...any) error
}

func NewTransactionRepository(db *dbpg.DB, topic ...string) TransactionRepository {
	kafkaTopic := events.DefaultTransactionsTopic
	if len(topic) > 0 && strings.TrimSpace(topic[0]) != "" {
		kafkaTopic = strings.TrimSpace(topic[0])
	}
	return &transactionRepository{db: db, topic: kafkaTopic}
}

func (r *transactionRepository) Create(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	dbTx, err := r.db.Master.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin create transaction tx: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }()

	query := `
		INSERT INTO transactions (
			user_id, amount, currency, from_account_id, to_account_id,
			provider_id, category_id, type, status, description,
			external_id, occurred_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`

	var id string
	var createdAt, updatedAt time.Time

	err = dbTx.QueryRowContext(
		ctx,
		query,
		tx.UserID, tx.Amount, tx.Currency, tx.FromAccountID, tx.ToAccountID,
		tx.ProviderID, tx.CategoryID, tx.Type, tx.Status, tx.Description,
		tx.ExternalID, tx.OccurredAt,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	tx.ID = id
	tx.CreatedAt = createdAt
	tx.UpdatedAt = updatedAt

	env, err := events.NewTransactionCreatedEnvelope(tx, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("build transaction.created outbox event: %w", err)
	}
	payload, err := events.MarshalTransactionEventEnvelope(env)
	if err != nil {
		return nil, fmt.Errorf("marshal transaction.created outbox event: %w", err)
	}
	if err := r.insertOutboxEvent(ctx, dbTx, env, tx.UserID, payload); err != nil {
		return nil, err
	}

	if err := dbTx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create transaction tx: %w", err)
	}
	return tx, nil
}

func (r *transactionRepository) GetByID(ctx context.Context, id, userID string) (*models.Transaction, error) {
	query := `
		SELECT id, user_id, amount, currency, from_account_id, to_account_id,
		       provider_id, category_id, type, status, description, external_id,
		       occurred_at, created_at, updated_at, deleted_at
		FROM transactions
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	var tx models.Transaction
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&tx.ID, &tx.UserID, &tx.Amount, &tx.Currency,
		&tx.FromAccountID, &tx.ToAccountID, &tx.ProviderID, &tx.CategoryID,
		&tx.Type, &tx.Status, &tx.Description, &tx.ExternalID,
		&tx.OccurredAt, &tx.CreatedAt, &tx.UpdatedAt, &tx.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &tx, nil
}

func (r *transactionRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*models.Transaction, error) {
	query := `
		SELECT id, user_id, amount, currency, from_account_id, to_account_id,
		       provider_id, category_id, type, status, description, external_id,
		       occurred_at, created_at, updated_at, deleted_at
		FROM transactions
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY occurred_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	defer rows.Close()

	var txs []*models.Transaction
	for rows.Next() {
		var tx models.Transaction
		err := rows.Scan(
			&tx.ID, &tx.UserID, &tx.Amount, &tx.Currency,
			&tx.FromAccountID, &tx.ToAccountID, &tx.ProviderID, &tx.CategoryID,
			&tx.Type, &tx.Status, &tx.Description, &tx.ExternalID,
			&tx.OccurredAt, &tx.CreatedAt, &tx.UpdatedAt, &tx.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		txs = append(txs, &tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return txs, nil
}

func (r *transactionRepository) Update(ctx context.Context, tx *models.Transaction) error {
	dbTx, err := r.db.Master.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin update transaction tx: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }()

	before, err := r.getByIDForUpdate(ctx, dbTx, tx.ID, tx.UserID)
	if err != nil {
		return err
	}

	query := `
		UPDATE transactions
		SET amount = $1, currency = $2, from_account_id = $3, to_account_id = $4,
		    provider_id = $5, category_id = $6, type = $7, status = $8,
		    description = $9, external_id = $10, occurred_at = $11, updated_at = NOW()
		WHERE id = $12 AND user_id = $13 AND deleted_at IS NULL
		RETURNING id, user_id, amount::text, currency, from_account_id, to_account_id,
		          provider_id, category_id, type, status, description, external_id,
		          occurred_at, created_at, updated_at, deleted_at
	`

	after, err := scanTransaction(dbTx.QueryRowContext(
		ctx,
		query,
		tx.Amount, tx.Currency, tx.FromAccountID, tx.ToAccountID,
		tx.ProviderID, tx.CategoryID, tx.Type, tx.Status,
		tx.Description, tx.ExternalID, tx.OccurredAt, tx.ID, tx.UserID,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperrors.ErrNotFound
		}
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	env, err := events.NewTransactionUpdatedEnvelope(before, after, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("build transaction.updated outbox event: %w", err)
	}
	payload, err := events.MarshalTransactionEventEnvelope(env)
	if err != nil {
		return fmt.Errorf("marshal transaction.updated outbox event: %w", err)
	}
	if err := r.insertOutboxEvent(ctx, dbTx, env, after.UserID, payload); err != nil {
		return err
	}

	if before.Status != after.Status {
		statusEnv, err := events.NewTransactionStatusChangedEnvelope(before, after, time.Now().UTC())
		if err != nil {
			return fmt.Errorf("build transaction.status_changed outbox event: %w", err)
		}
		statusPayload, err := events.MarshalTransactionEventEnvelope(statusEnv)
		if err != nil {
			return fmt.Errorf("marshal transaction.status_changed outbox event: %w", err)
		}
		if err := r.insertOutboxEvent(ctx, dbTx, statusEnv, after.UserID, statusPayload); err != nil {
			return err
		}
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit update transaction tx: %w", err)
	}
	return nil
}

func (r *transactionRepository) Delete(ctx context.Context, id, userID string) error {
	dbTx, err := r.db.Master.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin delete transaction tx: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }()

	tx, err := r.getByIDForUpdate(ctx, dbTx, id, userID)
	if err != nil {
		return err
	}

	query := `
		UPDATE transactions
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		RETURNING deleted_at, updated_at
	`

	err = dbTx.QueryRowContext(ctx, query, id, userID).Scan(&tx.DeletedAt, &tx.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperrors.ErrNotFound
		}
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	env, err := events.NewTransactionDeletedEnvelope(tx, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("build transaction.deleted outbox event: %w", err)
	}
	payload, err := events.MarshalTransactionEventEnvelope(env)
	if err != nil {
		return fmt.Errorf("marshal transaction.deleted outbox event: %w", err)
	}
	if err := r.insertOutboxEvent(ctx, dbTx, env, tx.UserID, payload); err != nil {
		return err
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit delete transaction tx: %w", err)
	}
	return nil
}

func (r *transactionRepository) getByIDForUpdate(ctx context.Context, tx *sql.Tx, id, userID string) (*models.Transaction, error) {
	const query = `
		SELECT id, user_id, amount::text, currency, from_account_id, to_account_id,
		       provider_id, category_id, type, status, description, external_id,
		       occurred_at, created_at, updated_at, deleted_at
		FROM transactions
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		FOR UPDATE
	`
	model, err := scanTransaction(tx.QueryRowContext(ctx, query, id, userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to lock transaction: %w", err)
	}
	return model, nil
}

func (r *transactionRepository) insertOutboxEvent(
	ctx context.Context,
	tx *sql.Tx,
	env *events.TransactionEventEnvelope,
	messageKey string,
	payload []byte,
) error {
	const outboxQuery = `
		INSERT INTO event_outbox (event_id, aggregate_id, event_type, topic, message_key, payload)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	if _, err := tx.ExecContext(
		ctx,
		outboxQuery,
		env.EventID,
		env.AggregateID,
		env.EventType,
		r.topic,
		messageKey,
		string(payload),
	); err != nil {
		return fmt.Errorf("failed to create outbox event: %w", err)
	}
	return nil
}

func scanTransaction(scanner transactionScanner) (*models.Transaction, error) {
	var tx models.Transaction
	err := scanner.Scan(
		&tx.ID, &tx.UserID, &tx.Amount, &tx.Currency,
		&tx.FromAccountID, &tx.ToAccountID, &tx.ProviderID, &tx.CategoryID,
		&tx.Type, &tx.Status, &tx.Description, &tx.ExternalID,
		&tx.OccurredAt, &tx.CreatedAt, &tx.UpdatedAt, &tx.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}
