package repository

import (
	"context"
	"fmt"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	"github.com/wb-go/wbf/dbpg"
)

type auditRepository struct {
	db *dbpg.DB
}

func NewAuditRepository(db *dbpg.DB) AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Create(ctx context.Context, log *models.AuditLog) error {
	const q = `
		INSERT INTO audit_logs (
			actor_id, actor_role, action, entity_type, entity_id, result, request_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	if _, err := r.db.Master.ExecContext(
		ctx,
		q,
		log.ActorID,
		log.ActorRole,
		log.Action,
		log.EntityType,
		log.EntityID,
		log.Result,
		log.RequestID,
	); err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}
