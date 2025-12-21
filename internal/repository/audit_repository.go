package repository

import (
	"context"
	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AuditRepository interface {
	Create(ctx context.Context, log models.AuditLog) error
	List(ctx context.Context, limit, offset int64) ([]models.AuditLog, error)
}

type mongoAuditRepository struct {
	Collection *mongo.Collection
}

func NewAuditRepository(db *mongo.Database) AuditRepository {
	return &mongoAuditRepository{
		Collection: db.Collection("audit_logs"),
	}
}

func (r *mongoAuditRepository) Create(ctx context.Context, log models.AuditLog) error {
	_, err := r.Collection.InsertOne(ctx, log)
	return err
}

func (r *mongoAuditRepository) List(ctx context.Context, limit, offset int64) ([]models.AuditLog, error) {
	opts := options.Find().SetLimit(limit).SetSkip(offset).SetSort(bson.M{"timestamp": -1})
	cursor, err := r.Collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	var logs []models.AuditLog
	if err = cursor.All(ctx, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}
