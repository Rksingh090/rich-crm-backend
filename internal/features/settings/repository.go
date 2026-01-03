package settings

import (
	"context"
	"fmt"

	"go-crm/internal/common/models"
	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SettingsRepository interface {
	GetByType(ctx context.Context, sType SettingsType) (*Settings, error)
	Upsert(ctx context.Context, settings *Settings) error
}

type SettingsRepositoryImpl struct {
	Collection *mongo.Collection
}

func NewSettingsRepository(mongodb *database.MongodbDB) SettingsRepository {
	return &SettingsRepositoryImpl{
		Collection: mongodb.DB.Collection("settings"),
	}
}

func (r *SettingsRepositoryImpl) GetByType(ctx context.Context, sType SettingsType) (*Settings, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	var settings Settings
	err = r.Collection.FindOne(ctx, bson.M{"type": sType, "tenant_id": oid}).Decode(&settings)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &settings, nil
}

func (r *SettingsRepositoryImpl) Upsert(ctx context.Context, settings *Settings) error {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}
	settings.TenantID = oid

	filter := bson.M{"type": settings.Type, "tenant_id": oid}
	update := bson.M{"$set": settings}
	opts := options.Update().SetUpsert(true)
	_, err = r.Collection.UpdateOne(ctx, filter, update, opts)
	return err
}
