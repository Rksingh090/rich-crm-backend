package settings

import (
	"context"

	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
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
	var settings Settings
	err := r.Collection.FindOne(ctx, bson.M{"type": sType}).Decode(&settings)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &settings, nil
}

func (r *SettingsRepositoryImpl) Upsert(ctx context.Context, settings *Settings) error {
	filter := bson.M{"type": settings.Type}
	update := bson.M{"$set": settings}
	opts := options.Update().SetUpsert(true)
	_, err := r.Collection.UpdateOne(ctx, filter, update, opts)
	return err
}
