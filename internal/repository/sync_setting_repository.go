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

type SyncSettingRepository interface {
	Create(ctx context.Context, setting *models.SyncSetting) error
	Get(ctx context.Context, id string) (*models.SyncSetting, error)
	List(ctx context.Context) ([]models.SyncSetting, error)
	ListActive(ctx context.Context) ([]models.SyncSetting, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
}

type SyncSettingRepositoryImpl struct {
	collection *mongo.Collection
}

func NewSyncSettingRepository(db *database.MongodbDB) SyncSettingRepository {
	return &SyncSettingRepositoryImpl{
		collection: db.DB.Collection("sync_settings"),
	}
}

func (r *SyncSettingRepositoryImpl) Create(ctx context.Context, setting *models.SyncSetting) error {
	if setting.ID.IsZero() {
		setting.ID = primitive.NewObjectID()
	}
	setting.CreatedAt = time.Now()
	setting.UpdatedAt = time.Now()
	if setting.LastSyncAt.IsZero() {
		setting.LastSyncAt = time.Time{} // Ensure it's not nil but a zero time
	}

	_, err := r.collection.InsertOne(ctx, setting)
	return err
}

func (r *SyncSettingRepositoryImpl) Get(ctx context.Context, id string) (*models.SyncSetting, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var setting models.SyncSetting
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&setting)
	if err != nil {
		return nil, err
	}

	return &setting, nil
}

func (r *SyncSettingRepositoryImpl) List(ctx context.Context) ([]models.SyncSetting, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var settings []models.SyncSetting
	if err = cursor.All(ctx, &settings); err != nil {
		return nil, err
	}

	return settings, nil
}

func (r *SyncSettingRepositoryImpl) ListActive(ctx context.Context) ([]models.SyncSetting, error) {
	filter := bson.M{"is_active": true}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var settings []models.SyncSetting
	if err = cursor.All(ctx, &settings); err != nil {
		return nil, err
	}

	return settings, nil
}

func (r *SyncSettingRepositoryImpl) Update(ctx context.Context, id string, updates map[string]interface{}) error {
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

func (r *SyncSettingRepositoryImpl) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
