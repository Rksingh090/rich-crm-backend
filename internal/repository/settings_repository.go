package repository

import (
	"context"

	"go-crm/internal/database"
	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SettingsRepository interface {
	GetByType(ctx context.Context, sType models.SettingsType) (*models.Settings, error)
	Upsert(ctx context.Context, settings *models.Settings) error
}

type SettingsRepositoryImpl struct {
	Collection *mongo.Collection
}

func NewSettingsRepository(mongodb *database.MongodbDB) SettingsRepository {
	return &SettingsRepositoryImpl{
		Collection: mongodb.DB.Collection("settings"),
	}
}

func (r *SettingsRepositoryImpl) GetByType(ctx context.Context, sType models.SettingsType) (*models.Settings, error) {
	var settings models.Settings
	err := r.Collection.FindOne(ctx, bson.M{"type": sType}).Decode(&settings)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Return nil if not found, not error
		}
		return nil, err
	}
	return &settings, nil
}

func (r *SettingsRepositoryImpl) Upsert(ctx context.Context, settings *models.Settings) error {
	filter := bson.M{"type": settings.Type}
	update := bson.M{"$set": settings}
	opts := options.Update().SetUpsert(true)
	_, err := r.Collection.UpdateOne(ctx, filter, update, opts)
	return err
}
