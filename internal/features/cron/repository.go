package cron_feature

import (
	"context"
	"fmt"
	"go-crm/internal/database"
	"time"

	common_models "go-crm/internal/common/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CronRepository interface {
	Create(ctx context.Context, cronJob *CronJob) error
	GetByID(ctx context.Context, id string) (*CronJob, error)
	List(ctx context.Context, filter map[string]interface{}) ([]CronJob, error)
	Update(ctx context.Context, cronJob *CronJob) error
	Delete(ctx context.Context, id string) error
	GetActive(ctx context.Context) ([]CronJob, error)
	UpdateLastRun(ctx context.Context, id string, lastRun time.Time, nextRun *time.Time) error

	// Log operations
	CreateLog(ctx context.Context, log *CronJobLog) error
	GetLogs(ctx context.Context, cronJobID string, limit int) ([]CronJobLog, error)
	UpdateLog(ctx context.Context, log *CronJobLog) error
}

type CronRepositoryImpl struct {
	DB *database.MongodbDB
}

func NewCronRepository(db *database.MongodbDB) CronRepository {
	return &CronRepositoryImpl{
		DB: db,
	}
}

func (r *CronRepositoryImpl) Create(ctx context.Context, cronJob *CronJob) error {
	tenantID, ok := ctx.Value(common_models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}

	cronJob.ID = primitive.NewObjectID()
	cronJob.TenantID = oid
	cronJob.CreatedAt = time.Now()
	cronJob.UpdatedAt = time.Now()

	collection := r.DB.GetTenantDB(tenantID).Collection("cron_jobs")
	_, err = collection.InsertOne(ctx, cronJob)
	return err
}

func (r *CronRepositoryImpl) GetByID(ctx context.Context, id string) (*CronJob, error) {
	tenantID, ok := ctx.Value(common_models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	collection := r.DB.GetTenantDB(tenantID).Collection("cron_jobs")
	var cronJob CronJob
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&cronJob)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &cronJob, nil
}

func (r *CronRepositoryImpl) List(ctx context.Context, filter map[string]interface{}) ([]CronJob, error) {
	tenantID, ok := ctx.Value(common_models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}

	var cronJobs []CronJob

	collection := r.DB.GetTenantDB(tenantID).Collection("cron_jobs")
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &cronJobs); err != nil {
		return nil, err
	}

	if cronJobs == nil {
		cronJobs = []CronJob{}
	}

	return cronJobs, nil
}

func (r *CronRepositoryImpl) Update(ctx context.Context, cronJob *CronJob) error {
	tenantID, ok := ctx.Value(common_models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("tenant context missing")
	}

	cronJob.UpdatedAt = time.Now()
	filter := bson.M{"_id": cronJob.ID}
	update := bson.M{"$set": cronJob}

	collection := r.DB.GetTenantDB(tenantID).Collection("cron_jobs")
	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *CronRepositoryImpl) Delete(ctx context.Context, id string) error {
	tenantID, ok := ctx.Value(common_models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("tenant context missing")
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	collection := r.DB.GetTenantDB(tenantID).Collection("cron_jobs")
	_, err = collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *CronRepositoryImpl) GetActive(ctx context.Context) ([]CronJob, error) {
	filter := bson.M{"active": true}
	return r.List(ctx, filter)
}

func (r *CronRepositoryImpl) UpdateLastRun(ctx context.Context, id string, lastRun time.Time, nextRun *time.Time) error {
	tenantID, ok := ctx.Value(common_models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("tenant context missing")
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"last_run":   lastRun,
			"next_run":   nextRun,
			"updated_at": time.Now(),
		},
	}

	collection := r.DB.GetTenantDB(tenantID).Collection("cron_jobs")
	_, err = collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *CronRepositoryImpl) CreateLog(ctx context.Context, log *CronJobLog) error {
	tenantID, ok := ctx.Value(common_models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}

	log.ID = primitive.NewObjectID()
	log.TenantID = oid
	log.CreatedAt = time.Now()

	logCollection := r.DB.GetTenantDB(tenantID).Collection("cron_job_logs")
	_, err = logCollection.InsertOne(ctx, log)
	return err
}

func (r *CronRepositoryImpl) GetLogs(ctx context.Context, cronJobID string, limit int) ([]CronJobLog, error) {
	tenantID, ok := ctx.Value(common_models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}

	objectID, err := primitive.ObjectIDFromHex(cronJobID)
	if err != nil {
		return nil, err
	}

	var logs []CronJobLog

	logCollection := r.DB.GetTenantDB(tenantID).Collection("cron_job_logs")
	opts := options.Find().
		SetSort(bson.D{{Key: "start_time", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := logCollection.Find(ctx, bson.M{"cron_job_id": objectID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &logs); err != nil {
		return nil, err
	}

	if logs == nil {
		logs = []CronJobLog{}
	}

	return logs, nil
}

func (r *CronRepositoryImpl) UpdateLog(ctx context.Context, log *CronJobLog) error {
	tenantID, ok := ctx.Value(common_models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("tenant context missing")
	}

	filter := bson.M{"_id": log.ID}
	update := bson.M{"$set": log}

	logCollection := r.DB.GetTenantDB(tenantID).Collection("cron_job_logs")
	_, err := logCollection.UpdateOne(ctx, filter, update)
	return err
}
