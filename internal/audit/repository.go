package audit

import (
	"context"
	"log"

	"team-management/internal/models"
	"team-management/internal/utils"

	"github.com/jmoiron/sqlx"
)

type AuditRepository interface {
	CreateLog(ctx context.Context, logEntry *models.AuditLog) error
}

type auditRepoImpl struct {
	db *sqlx.DB
}

func NewAuditRepository(db *sqlx.DB) AuditRepository {
	return &auditRepoImpl{db: db}
}

func (r *auditRepoImpl) CreateLog(ctx context.Context, logEntry *models.AuditLog) error {
	query := `
        INSERT INTO audit_logs (user_id, action, entity_type, entity_id, details) 
        VALUES (?, ?, ?, ?, ?)
    `

	_, err := r.db.ExecContext(ctx, query,
		logEntry.UserID,
		logEntry.Action,
		logEntry.EntityType,
		logEntry.EntityID,
		logEntry.Details,
	)

	if err != nil {
		log.Printf("Database Error: Failed to insert audit log: %v", err)
		return utils.NewInternalError("Failed to create audit log", err)
	}

	return nil
}
