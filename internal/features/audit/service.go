package audit

import (
	"context"
	common_models "go-crm/internal/common/models"
	"go-crm/pkg/utils"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserFinder interface {
	FindByIDs(ctx context.Context, ids []string) ([]common_models.User, error)
}

type AuditService interface {
	LogChange(ctx context.Context, action common_models.AuditAction, module string, recordID string, changes map[string]common_models.Change) error
	ListLogs(ctx context.Context, filters map[string]interface{}, page, limit int64) ([]common_models.AuditLog, error)
}

type AuditServiceImpl struct {
	Repo     AuditRepository
	UserRepo UserFinder
}

func NewAuditService(repo AuditRepository, userRepo UserFinder) AuditService {
	return &AuditServiceImpl{
		Repo:     repo,
		UserRepo: userRepo,
	}
}

func (s *AuditServiceImpl) LogChange(ctx context.Context, action common_models.AuditAction, module string, recordID string, changes map[string]common_models.Change) error {
	// Extract Actor from Context
	actorID := "system"
	if claims, ok := ctx.Value(utils.UserClaimsKey).(*utils.UserClaims); ok {
		actorID = claims.UserID
	}

	log := common_models.AuditLog{
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

func (s *AuditServiceImpl) ListLogs(ctx context.Context, filters map[string]interface{}, page, limit int64) ([]common_models.AuditLog, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit
	logs, err := s.Repo.List(ctx, filters, limit, offset)
	if err != nil {
		return nil, err
	}

	// Collect Actor IDs
	actorIDs := make([]string, 0)
	uniqueIDs := make(map[string]bool)
	for _, log := range logs {
		if log.ActorID != "system" && log.ActorID != "" {
			if !uniqueIDs[log.ActorID] {
				uniqueIDs[log.ActorID] = true
				actorIDs = append(actorIDs, log.ActorID)
			}
		}
	}

	// Batch Fetch Users
	userMap := make(map[string]string)
	if len(actorIDs) > 0 {
		users, err := s.UserRepo.FindByIDs(ctx, actorIDs)
		if err == nil {
			for _, user := range users {
				userMap[user.ID.Hex()] = user.Username
			}
		}
	}

	// Populate Actor Names
	for i, log := range logs {
		if log.ActorID == "system" || log.ActorID == "" {
			logs[i].ActorName = "System"
		} else {
			if name, ok := userMap[log.ActorID]; ok {
				logs[i].ActorName = name
			} else {
				logs[i].ActorName = "Unknown User"
			}
		}
	}

	return logs, nil
}
