package sync

import (
	"context"
	"time"

	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SyncSettingRepository interface {
	Create(ctx context.Context, setting *SyncSetting) error
	Get(ctx context.Context, id string) (*SyncSetting, error)
	List(ctx context.Context) ([]SyncSetting, error)
	ListActive(ctx context.Context) ([]SyncSetting, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
}

type SyncLogRepository interface {
	Create(ctx context.Context, log *SyncLog) error
	GetLatest(ctx context.Context, settingID string) (*SyncLog, error)
	List(ctx context.Context, settingID string, limit int64) ([]SyncLog, error)
	Update(ctx context.Context, log *SyncLog) error
}

type SyncSettingRepositoryImpl struct {
	collection *mongo.Collection
}

func NewSyncSettingRepository(db *database.MongodbDB) SyncSettingRepository {
	return &SyncSettingRepositoryImpl{
		collection: db.DB.Collection("sync_settings"),
	}
}

func (r *SyncSettingRepositoryImpl) Create(ctx context.Context, setting *SyncSetting) error {
	if setting.ID.IsZero() {
		setting.ID = primitive.NewObjectID()
	}
	setting.CreatedAt = time.Now()
	setting.UpdatedAt = time.Now()
	if setting.LastSyncAt.IsZero() {
		setting.LastSyncAt = time.Time{}
	}

	_, err := r.collection.InsertOne(ctx, setting)
	return err
}

func (r *SyncSettingRepositoryImpl) Get(ctx context.Context, id string) (*SyncSetting, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var setting SyncSetting
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&setting)
	if err != nil {
		return nil, err
	}

	return &setting, nil
}

func (r *SyncSettingRepositoryImpl) List(ctx context.Context) ([]SyncSetting, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var settings []SyncSetting
	if err = cursor.All(ctx, &settings); err != nil {
		return nil, err
	}

	return settings, nil
}

func (r *SyncSettingRepositoryImpl) ListActive(ctx context.Context) ([]SyncSetting, error) {
	filter := bson.M{"is_active": true}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var settings []SyncSetting
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

type SyncLogRepositoryImpl struct {
	collection *mongo.Collection
}

func NewSyncLogRepository(db *database.MongodbDB) SyncLogRepository {
	return &SyncLogRepositoryImpl{
		collection: db.DB.Collection("sync_logs"),
	}
}

func (r *SyncLogRepositoryImpl) Create(ctx context.Context, log *SyncLog) error {
	if log.ID.IsZero() {
		log.ID = primitive.NewObjectID()
	}
	if log.StartTime.IsZero() {
		log.StartTime = time.Now()
	}

	_, err := r.collection.InsertOne(ctx, log)
	return err
}

func (r *SyncLogRepositoryImpl) GetLatest(ctx context.Context, settingID string) (*SyncLog, error) {
	oid, err := primitive.ObjectIDFromHex(settingID)
	if err != nil {
		return nil, err
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "start_time", Value: -1}})
	var log SyncLog
	err = r.collection.FindOne(ctx, bson.M{"sync_setting_id": oid}, opts).Decode(&log)
	if err != nil {
		return nil, err
	}

	return &log, nil
}

func (r *SyncLogRepositoryImpl) List(ctx context.Context, settingID string, limit int64) ([]SyncLog, error) {
	oid, err := primitive.ObjectIDFromHex(settingID)
	if err != nil {
		return nil, err
	}

	opts := options.Find().SetSort(bson.D{{Key: "start_time", Value: -1}}).SetLimit(limit)
	cursor, err := r.collection.Find(ctx, bson.M{"sync_setting_id": oid}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []SyncLog
	if err = cursor.All(ctx, &logs); err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *SyncLogRepositoryImpl) Update(ctx context.Context, log *SyncLog) error {
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": log.ID}, log)
	return err
}
