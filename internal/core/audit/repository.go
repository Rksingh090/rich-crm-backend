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
	DB *database.MongodbDB
}

func NewAuditRepository(mongodb *database.MongodbDB) AuditRepository {
	return &AuditRepositoryImpl{
		DB: mongodb,
	}
}

func (r *AuditRepositoryImpl) getCollection(ctx context.Context) *mongo.Collection {
	tenantID, ok := ctx.Value(common_models.TenantIDKey).(string)
	if ok && tenantID != "" {
		return r.DB.GetTenantDB(tenantID).Collection("audit_logs")
	}
	return r.DB.GetControlPlaneDB().Collection("audit_logs")
}

func (r *AuditRepositoryImpl) Create(ctx context.Context, log common_models.AuditLog) error {
	coll := r.getCollection(ctx)

	tenantID, ok := ctx.Value(common_models.TenantIDKey).(string)
	if ok && tenantID != "" {
		if oid, err := primitive.ObjectIDFromHex(tenantID); err == nil {
			log.TenantID = oid
		}
	}

	_, err := coll.InsertOne(ctx, log)
	return err
}

func (r *AuditRepositoryImpl) List(ctx context.Context, filters map[string]interface{}, limit, offset int64) ([]common_models.AuditLog, error) {
	coll := r.getCollection(ctx)
	opts := options.Find().SetLimit(limit).SetSkip(offset).SetSort(bson.M{"timestamp": -1})

	query := bson.M{}

	// If in Tenant DB, we don't strictly need to filter by tenant_id as DB is isolated,
	// but it doesn't hurt and ensures consistency if fallback to global.
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

	cursor, err := coll.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	var logs []common_models.AuditLog
	if err = cursor.All(ctx, &logs); err != nil {
		return nil, err
	}
	// Initializing slice to avoid returning null in JSON
	if logs == nil {
		logs = []common_models.AuditLog{}
	}
	return logs, nil
}
