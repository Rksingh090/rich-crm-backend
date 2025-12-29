package repository

import (
	"context"
	"go-crm/internal/database"
	"go-crm/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ImportRepository interface {
	Create(ctx context.Context, job *models.ImportJob) error
	Get(ctx context.Context, id string) (*models.ImportJob, error)
	Update(ctx context.Context, id string, job *models.ImportJob) error
	FindByUserID(ctx context.Context, userID string, limit int) ([]models.ImportJob, error)
	UpdateStatus(ctx context.Context, id string, status models.ImportStatus) error
}

type ImportRepositoryImpl struct {
	collection *mongo.Collection
}

func NewImportRepository(db *database.MongodbDB) ImportRepository {
	return &ImportRepositoryImpl{
		collection: db.DB.Collection("import_jobs"),
	}
}

func (r *ImportRepositoryImpl) Create(ctx context.Context, job *models.ImportJob) error {
	if job.ID.IsZero() {
		job.ID = primitive.NewObjectID()
	}
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	job.Status = models.ImportStatusPending

	_, err := r.collection.InsertOne(ctx, job)
	return err
}

func (r *ImportRepositoryImpl) Get(ctx context.Context, id string) (*models.ImportJob, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var job models.ImportJob
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&job)
	if err != nil {
		return nil, err
	}

	return &job, nil
}

func (r *ImportRepositoryImpl) Update(ctx context.Context, id string, job *models.ImportJob) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	job.UpdatedAt = time.Now()
	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objID}, job)
	return err
}

func (r *ImportRepositoryImpl) FindByUserID(ctx context.Context, userID string, limit int) ([]models.ImportJob, error) {
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"user_id": objID}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jobs []models.ImportJob
	if err = cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (r *ImportRepositoryImpl) UpdateStatus(ctx context.Context, id string, status models.ImportStatus) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	if status == models.ImportStatusCompleted || status == models.ImportStatusFailed {
		now := time.Now()
		update = bson.M{
			"$set": bson.M{
				"status":       status,
				"updated_at":   time.Now(),
				"completed_at": &now,
			},
		}
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}
