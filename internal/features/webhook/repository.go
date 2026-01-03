package webhook

import (
	"context"
	"time"

	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WebhookRepository interface {
	Create(ctx context.Context, webhook *Webhook) error
	Get(ctx context.Context, id string) (*Webhook, error)
	List(ctx context.Context) ([]Webhook, error)
	ListByEvent(ctx context.Context, event string) ([]Webhook, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
}

type WebhookLogRepository interface {
	Create(ctx context.Context, log *WebhookLog) error
	ListByWebhookID(ctx context.Context, webhookID string) ([]WebhookLog, error)
}

type WebhookRepositoryImpl struct {
	collection *mongo.Collection
}

func NewWebhookRepository(db *database.MongodbDB) WebhookRepository {
	return &WebhookRepositoryImpl{
		collection: db.DB.Collection("webhooks"),
	}
}

func (r *WebhookRepositoryImpl) Create(ctx context.Context, webhook *Webhook) error {
	if webhook.ID.IsZero() {
		webhook.ID = primitive.NewObjectID()
	}
	webhook.CreatedAt = time.Now()
	webhook.UpdatedAt = time.Now()
	webhook.IsActive = true // Default to true

	_, err := r.collection.InsertOne(ctx, webhook)
	return err
}

func (r *WebhookRepositoryImpl) Get(ctx context.Context, id string) (*Webhook, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&webhook)
	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

func (r *WebhookRepositoryImpl) List(ctx context.Context) ([]Webhook, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var webhooks []Webhook
	if err = cursor.All(ctx, &webhooks); err != nil {
		return nil, err
	}

	return webhooks, nil
}

func (r *WebhookRepositoryImpl) ListByEvent(ctx context.Context, event string) ([]Webhook, error) {
	filter := bson.M{
		"events":    event,
		"is_active": true,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var webhooks []Webhook
	if err = cursor.All(ctx, &webhooks); err != nil {
		return nil, err
	}

	return webhooks, nil
}

func (r *WebhookRepositoryImpl) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	updates["updated_at"] = time.Now()
	_, err = r.collection.UpdateOne(
		ctx,
		bson.M{"_id": oid},
		bson.M{"$set": updates},
	)
	return err
}

func (r *WebhookRepositoryImpl) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}

type WebhookLogRepositoryImpl struct {
	collection *mongo.Collection
}

func NewWebhookLogRepository(db *database.MongodbDB) WebhookLogRepository {
	return &WebhookLogRepositoryImpl{
		collection: db.DB.Collection("webhook_logs"),
	}
}

func (r *WebhookLogRepositoryImpl) Create(ctx context.Context, log *WebhookLog) error {
	if log.ID.IsZero() {
		log.ID = primitive.NewObjectID()
	}
	log.CreatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, log)
	return err
}

func (r *WebhookLogRepositoryImpl) ListByWebhookID(ctx context.Context, webhookID string) ([]WebhookLog, error) {
	oid, err := primitive.ObjectIDFromHex(webhookID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"webhook_id": oid}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(50)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []WebhookLog
	if err = cursor.All(ctx, &logs); err != nil {
		return nil, err
	}

	return logs, nil
}
