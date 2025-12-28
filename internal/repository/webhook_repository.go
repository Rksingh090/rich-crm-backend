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

type WebhookRepository interface {
	Create(ctx context.Context, webhook *models.Webhook) error
	Get(ctx context.Context, id string) (*models.Webhook, error)
	List(ctx context.Context) ([]models.Webhook, error)
	ListByEvent(ctx context.Context, event string) ([]models.Webhook, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
}

type WebhookRepositoryImpl struct {
	collection *mongo.Collection
}

func NewWebhookRepository(db *database.MongodbDB) WebhookRepository {
	return &WebhookRepositoryImpl{
		collection: db.DB.Collection("webhooks"),
	}
}

func (r *WebhookRepositoryImpl) Create(ctx context.Context, webhook *models.Webhook) error {
	if webhook.ID.IsZero() {
		webhook.ID = primitive.NewObjectID()
	}
	webhook.CreatedAt = time.Now()
	webhook.UpdatedAt = time.Now()
	webhook.IsActive = true // Default to true

	_, err := r.collection.InsertOne(ctx, webhook)
	return err
}

func (r *WebhookRepositoryImpl) Get(ctx context.Context, id string) (*models.Webhook, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var webhook models.Webhook
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&webhook)
	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

func (r *WebhookRepositoryImpl) List(ctx context.Context) ([]models.Webhook, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var webhooks []models.Webhook
	if err = cursor.All(ctx, &webhooks); err != nil {
		return nil, err
	}

	return webhooks, nil
}

func (r *WebhookRepositoryImpl) ListByEvent(ctx context.Context, event string) ([]models.Webhook, error) {
	// Find webhooks where 'events' array contains 'event'
	filter := bson.M{
		"events":    event,
		"is_active": true,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var webhooks []models.Webhook
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
