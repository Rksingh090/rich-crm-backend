package service

import (
	"context"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"go-crm/pkg/utils"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuditServiceImpl struct {
	Repo repository.AuditRepository
}

func NewAuditServiceImpl(repo repository.AuditRepository) *AuditServiceImpl {
	return &AuditServiceImpl{Repo: repo}
}

func (s *AuditServiceImpl) LogChange(ctx context.Context, action models.AuditAction, module string, recordID string, changes map[string]models.Change) error {
	// Extract Actor from Context
	actorID := "system"
	if claims, ok := ctx.Value(utils.UserClaimsKey).(*utils.UserClaims); ok {
		actorID = claims.UserID
	}

	log := models.AuditLog{
		ID:        primitive.NewObjectID(),
		Action:    action,
		Module:    module,
		RecordID:  recordID,
		ActorID:   actorID,
		Changes:   changes,
		Timestamp: time.Now(),
	}

	return s.Repo.Create(ctx, log)
}

func (s *AuditServiceImpl) ListLogs(ctx context.Context, page, limit int64) ([]models.AuditLog, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit
	return s.Repo.List(ctx, limit, offset)
}
