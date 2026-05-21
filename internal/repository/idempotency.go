package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	apperrors "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/errors"
	"github.com/wb-go/wbf/dbpg"
)

type idempotencyRepository struct {
	db *dbpg.DB
}

func NewIdempotencyRepository(db *dbpg.DB) IdempotencyRepository {
	return &idempotencyRepository{db: db}
}

func (r *idempotencyRepository) Get(ctx context.Context, userID, idempotencyKey string) (*IdempotencyRecord, error) {
	const q = `
		SELECT body_hash, http_status, response_json
		FROM idempotency_keys
		WHERE user_id = $1 AND idempotency_key = $2
	`
	var rec IdempotencyRecord
	err := r.db.Master.QueryRowContext(ctx, q, userID, idempotencyKey).Scan(
		&rec.BodyHash, &rec.HTTPStatus, &rec.ResponseJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("idempotency get: %w", err)
	}
	return &rec, nil
}

func (r *idempotencyRepository) Save(ctx context.Context, userID, idempotencyKey string, bodyHash []byte, httpStatus int, responseJSON []byte) error {
	const q = `
		INSERT INTO idempotency_keys (user_id, idempotency_key, body_hash, http_status, response_json)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, idempotency_key) DO NOTHING
	`
	res, err := r.db.Master.ExecContext(ctx, q, userID, idempotencyKey, bodyHash, httpStatus, string(responseJSON))
	if err != nil {
		return fmt.Errorf("idempotency save: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil
	}
	return nil
}
