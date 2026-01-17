package webhook

import (
	"context"
	"fmt"
	"time"

	"go-crm/internal/common/models"
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
	DB *database.MongodbDB
}

func NewWebhookRepository(db *database.MongodbDB) WebhookRepository {
	return &WebhookRepositoryImpl{
		DB: db,
	}
}

func (r *WebhookRepositoryImpl) getCollection(ctx context.Context) (*mongo.Collection, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	return r.DB.GetTenantDB(tenantID).Collection("webhooks"), nil
}

func (r *WebhookRepositoryImpl) Create(ctx context.Context, webhook *Webhook) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	if webhook.ID.IsZero() {
		webhook.ID = primitive.NewObjectID()
	}

	// Populate TenantID and App from context
	if tenantID, ok := ctx.Value(models.TenantIDKey).(string); ok {
		if oid, err := primitive.ObjectIDFromHex(tenantID); err == nil {
			webhook.TenantID = oid
		}
	}
	if appID, ok := ctx.Value(models.AppIDKey).(string); ok {
		webhook.App = appID
	} else {
		webhook.App = string(models.AppCRM) // Default to CRM if not specified
	}

	webhook.CreatedAt = time.Now()
	webhook.UpdatedAt = time.Now()
	webhook.IsActive = true // Default to true

	_, err = coll.InsertOne(ctx, webhook)
	return err
}

func (r *WebhookRepositoryImpl) Get(ctx context.Context, id string) (*Webhook, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	err = coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&webhook)
	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

func (r *WebhookRepositoryImpl) List(ctx context.Context) ([]Webhook, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := coll.Find(ctx, bson.M{}, opts)
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
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"events":    event,
		"is_active": true,
	}

	cursor, err := coll.Find(ctx, filter)
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
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	updates["updated_at"] = time.Now()
	_, err = coll.UpdateOne(
		ctx,
		bson.M{"_id": oid},
		bson.M{"$set": updates},
	)
	return err
}

func (r *WebhookRepositoryImpl) Delete(ctx context.Context, id string) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = coll.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}

type WebhookLogRepositoryImpl struct {
	DB *database.MongodbDB
}

func NewWebhookLogRepository(db *database.MongodbDB) WebhookLogRepository {
	return &WebhookLogRepositoryImpl{
		DB: db,
	}
}

func (r *WebhookLogRepositoryImpl) getCollection(ctx context.Context) (*mongo.Collection, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	return r.DB.GetTenantDB(tenantID).Collection("webhook_logs"), nil
}

func (r *WebhookLogRepositoryImpl) Create(ctx context.Context, log *WebhookLog) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	if log.ID.IsZero() {
		log.ID = primitive.NewObjectID()
	}

	// Populate TenantID and App from context
	if tenantID, ok := ctx.Value(models.TenantIDKey).(string); ok {
		if oid, err := primitive.ObjectIDFromHex(tenantID); err == nil {
			log.TenantID = oid
		}
	}
	if appID, ok := ctx.Value(models.AppIDKey).(string); ok {
		log.App = appID
	} else {
		log.App = string(models.AppCRM) // Default to CRM if not specified
	}

	log.CreatedAt = time.Now()

	_, err = coll.InsertOne(ctx, log)
	return err
}

func (r *WebhookLogRepositoryImpl) ListByWebhookID(ctx context.Context, webhookID string) ([]WebhookLog, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	oid, err := primitive.ObjectIDFromHex(webhookID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"webhook_id": oid}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(50)

	cursor, err := coll.Find(ctx, filter, opts)
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
