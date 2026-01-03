package cron_feature

import (
	"context"
	"go-crm/internal/database"
	"time"

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
	collection    *mongo.Collection
	logCollection *mongo.Collection
}

func NewCronRepository(db *database.MongodbDB) CronRepository {
	return &CronRepositoryImpl{
		collection:    db.DB.Collection("cron_jobs"),
		logCollection: db.DB.Collection("cron_job_logs"),
	}
}

func (r *CronRepositoryImpl) Create(ctx context.Context, cronJob *CronJob) error {
	cronJob.ID = primitive.NewObjectID()
	cronJob.CreatedAt = time.Now()
	cronJob.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, cronJob)
	return err
}

func (r *CronRepositoryImpl) GetByID(ctx context.Context, id string) (*CronJob, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var cronJob CronJob
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&cronJob)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &cronJob, nil
}

func (r *CronRepositoryImpl) List(ctx context.Context, filter map[string]interface{}) ([]CronJob, error) {
	var cronJobs []CronJob

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
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
	cronJob.UpdatedAt = time.Now()
	filter := bson.M{"_id": cronJob.ID}
	update := bson.M{"$set": cronJob}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *CronRepositoryImpl) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *CronRepositoryImpl) GetActive(ctx context.Context) ([]CronJob, error) {
	filter := bson.M{"active": true}
	return r.List(ctx, filter)
}

func (r *CronRepositoryImpl) UpdateLastRun(ctx context.Context, id string, lastRun time.Time, nextRun *time.Time) error {
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

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *CronRepositoryImpl) CreateLog(ctx context.Context, log *CronJobLog) error {
	log.ID = primitive.NewObjectID()
	log.CreatedAt = time.Now()

	_, err := r.logCollection.InsertOne(ctx, log)
	return err
}

func (r *CronRepositoryImpl) GetLogs(ctx context.Context, cronJobID string, limit int) ([]CronJobLog, error) {
	objectID, err := primitive.ObjectIDFromHex(cronJobID)
	if err != nil {
		return nil, err
	}

	var logs []CronJobLog

	opts := options.Find().
		SetSort(bson.D{{Key: "start_time", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.logCollection.Find(ctx, bson.M{"cron_job_id": objectID}, opts)
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
	filter := bson.M{"_id": log.ID}
	update := bson.M{"$set": log}

	_, err := r.logCollection.UpdateOne(ctx, filter, update)
	return err
}
