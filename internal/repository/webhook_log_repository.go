package repository

import (
	"context"
	"time"

	"go-crm/internal/database"
	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WebhookLogRepository interface {
	Create(ctx context.Context, log *models.WebhookLog) error
	ListByWebhookID(ctx context.Context, webhookID string) ([]models.WebhookLog, error)
}

type WebhookLogRepositoryImpl struct {
	collection *mongo.Collection
}

func NewWebhookLogRepository(db *database.MongodbDB) WebhookLogRepository {
	return &WebhookLogRepositoryImpl{
		collection: db.DB.Collection("webhook_logs"),
	}
}

func (r *WebhookLogRepositoryImpl) Create(ctx context.Context, log *models.WebhookLog) error {
	if log.ID.IsZero() {
		log.ID = primitive.NewObjectID()
	}
	log.CreatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, log)
	return err
}

func (r *WebhookLogRepositoryImpl) ListByWebhookID(ctx context.Context, webhookID string) ([]models.WebhookLog, error) {
	oid, err := primitive.ObjectIDFromHex(webhookID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"webhook_id": oid}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(50) // Limit to last 50 logs

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []models.WebhookLog
	if err = cursor.All(ctx, &logs); err != nil {
		return nil, err
	}

	return logs, nil
}
