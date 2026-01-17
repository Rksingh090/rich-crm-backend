package sync

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
	DB *database.MongodbDB
}

func NewSyncSettingRepository(db *database.MongodbDB) SyncSettingRepository {
	return &SyncSettingRepositoryImpl{
		DB: db,
	}
}

func (r *SyncSettingRepositoryImpl) getCollection(ctx context.Context) (*mongo.Collection, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	return r.DB.GetTenantDB(tenantID).Collection("sync_settings"), nil
}

func (r *SyncSettingRepositoryImpl) Create(ctx context.Context, setting *SyncSetting) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	if setting.ID.IsZero() {
		setting.ID = primitive.NewObjectID()
	}

	// Populate TenantID and App from context
	if tenantID, ok := ctx.Value(models.TenantIDKey).(string); ok {
		if oid, err := primitive.ObjectIDFromHex(tenantID); err == nil {
			setting.TenantID = oid
		}
	}
	if appID, ok := ctx.Value(models.AppIDKey).(string); ok {
		setting.App = appID
	} else {
		setting.App = string(models.AppCRM) // Default to CRM if not specified
	}

	setting.CreatedAt = time.Now()
	setting.UpdatedAt = time.Now()
	if setting.LastSyncAt.IsZero() {
		setting.LastSyncAt = time.Time{}
	}

	_, err = coll.InsertOne(ctx, setting)
	return err
}

func (r *SyncSettingRepositoryImpl) Get(ctx context.Context, id string) (*SyncSetting, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var setting SyncSetting
	err = coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&setting)
	if err != nil {
		return nil, err
	}

	return &setting, nil
}

func (r *SyncSettingRepositoryImpl) List(ctx context.Context) ([]SyncSetting, error) {
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

	var settings []SyncSetting
	if err = cursor.All(ctx, &settings); err != nil {
		return nil, err
	}

	return settings, nil
}

func (r *SyncSettingRepositoryImpl) ListActive(ctx context.Context) ([]SyncSetting, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"is_active": true}
	cursor, err := coll.Find(ctx, filter)
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

func (r *SyncSettingRepositoryImpl) Delete(ctx context.Context, id string) error {
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

type SyncLogRepositoryImpl struct {
	DB *database.MongodbDB
}

func NewSyncLogRepository(db *database.MongodbDB) SyncLogRepository {
	return &SyncLogRepositoryImpl{
		DB: db,
	}
}

func (r *SyncLogRepositoryImpl) getCollection(ctx context.Context) (*mongo.Collection, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	return r.DB.GetTenantDB(tenantID).Collection("sync_logs"), nil
}

func (r *SyncLogRepositoryImpl) Create(ctx context.Context, log *SyncLog) error {
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

	if log.StartTime.IsZero() {
		log.StartTime = time.Now()
	}

	_, err = coll.InsertOne(ctx, log)
	return err
}

func (r *SyncLogRepositoryImpl) GetLatest(ctx context.Context, settingID string) (*SyncLog, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	oid, err := primitive.ObjectIDFromHex(settingID)
	if err != nil {
		return nil, err
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "start_time", Value: -1}})
	var log SyncLog
	err = coll.FindOne(ctx, bson.M{"sync_setting_id": oid}, opts).Decode(&log)
	if err != nil {
		return nil, err
	}

	return &log, nil
}

func (r *SyncLogRepositoryImpl) List(ctx context.Context, settingID string, limit int64) ([]SyncLog, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	oid, err := primitive.ObjectIDFromHex(settingID)
	if err != nil {
		return nil, err
	}

	opts := options.Find().SetSort(bson.D{{Key: "start_time", Value: -1}}).SetLimit(limit)
	cursor, err := coll.Find(ctx, bson.M{"sync_setting_id": oid}, opts)
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
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	_, err = coll.ReplaceOne(ctx, bson.M{"_id": log.ID}, log)
	return err
}
