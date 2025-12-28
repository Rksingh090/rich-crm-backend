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

type SyncLogRepository interface {
	Create(ctx context.Context, log *models.SyncLog) error
	GetLatest(ctx context.Context, settingID string) (*models.SyncLog, error)
	List(ctx context.Context, settingID string, limit int64) ([]models.SyncLog, error)
	Update(ctx context.Context, log *models.SyncLog) error
}

type SyncLogRepositoryImpl struct {
	collection *mongo.Collection
}

func NewSyncLogRepository(db *database.MongodbDB) SyncLogRepository {
	return &SyncLogRepositoryImpl{
		collection: db.DB.Collection("sync_logs"),
	}
}

func (r *SyncLogRepositoryImpl) Create(ctx context.Context, log *models.SyncLog) error {
	if log.ID.IsZero() {
		log.ID = primitive.NewObjectID()
	}
	if log.StartTime.IsZero() {
		log.StartTime = time.Now()
	}

	_, err := r.collection.InsertOne(ctx, log)
	return err
}

func (r *SyncLogRepositoryImpl) GetLatest(ctx context.Context, settingID string) (*models.SyncLog, error) {
	oid, err := primitive.ObjectIDFromHex(settingID)
	if err != nil {
		return nil, err
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "start_time", Value: -1}})
	var log models.SyncLog
	err = r.collection.FindOne(ctx, bson.M{"sync_setting_id": oid}, opts).Decode(&log)
	if err != nil {
		return nil, err
	}

	return &log, nil
}

func (r *SyncLogRepositoryImpl) List(ctx context.Context, settingID string, limit int64) ([]models.SyncLog, error) {
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

	var logs []models.SyncLog
	if err = cursor.All(ctx, &logs); err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *SyncLogRepositoryImpl) Update(ctx context.Context, log *models.SyncLog) error {
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": log.ID}, log)
	return err
}
