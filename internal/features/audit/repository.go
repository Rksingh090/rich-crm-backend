package audit

import (
	"context"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AuditRepository interface {
	Create(ctx context.Context, log common_models.AuditLog) error
	List(ctx context.Context, filters map[string]interface{}, limit, offset int64) ([]common_models.AuditLog, error)
}

type AuditRepositoryImpl struct {
	Collection *mongo.Collection
}

func NewAuditRepository(mongodb *database.MongodbDB) AuditRepository {
	return &AuditRepositoryImpl{
		Collection: mongodb.DB.Collection("audit_logs"),
	}
}

func (r *AuditRepositoryImpl) Create(ctx context.Context, log common_models.AuditLog) error {
	tenantID, ok := ctx.Value(common_models.TenantIDKey).(string)
	if ok && tenantID != "" {
		if oid, err := primitive.ObjectIDFromHex(tenantID); err == nil {
			log.TenantID = oid
		}
	}
	// Note: Audit logs might sometimes be created without tenant context (e.g. system events).
	// But mostly they should have it.

	_, err := r.Collection.InsertOne(ctx, log)
	return err
}

func (r *AuditRepositoryImpl) List(ctx context.Context, filters map[string]interface{}, limit, offset int64) ([]common_models.AuditLog, error) {
	opts := options.Find().SetLimit(limit).SetSkip(offset).SetSort(bson.M{"timestamp": -1})

	query := bson.M{}

	tenantID, ok := ctx.Value(common_models.TenantIDKey).(string)
	if ok && tenantID != "" {
		if oid, err := primitive.ObjectIDFromHex(tenantID); err == nil {
			query["tenant_id"] = oid
		}
	}

	for k, v := range filters {
		if v == nil {
			continue
		}
		if str, ok := v.(string); ok && str == "" {
			continue
		}
		query[k] = v
	}

	cursor, err := r.Collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	var logs []common_models.AuditLog
	if err = cursor.All(ctx, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}
