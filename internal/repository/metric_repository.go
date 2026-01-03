package repository

import (
	"context"
	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MetricRepository interface {
	Create(ctx context.Context, metric *models.Metric) error
	Get(ctx context.Context, id string) (*models.Metric, error)
	List(ctx context.Context) ([]models.Metric, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	FindByDataSource(ctx context.Context, dataSourceID string) ([]models.Metric, error)
	FindByModule(ctx context.Context, module string) ([]models.Metric, error)
}

type MetricRepositoryImpl struct {
	collection *mongo.Collection
}

func NewMetricRepository(db *mongo.Database) MetricRepository {
	return &MetricRepositoryImpl{
		collection: db.Collection("metrics"),
	}
}

func (r *MetricRepositoryImpl) Create(ctx context.Context, metric *models.Metric) error {
	metric.ID = primitive.NewObjectID()
	_, err := r.collection.InsertOne(ctx, metric)
	return err
}

func (r *MetricRepositoryImpl) Get(ctx context.Context, id string) (*models.Metric, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var metric models.Metric
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&metric)
	if err != nil {
		return nil, err
	}

	return &metric, nil
}

func (r *MetricRepositoryImpl) List(ctx context.Context) ([]models.Metric, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var metrics []models.Metric
	if err = cursor.All(ctx, &metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

func (r *MetricRepositoryImpl) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": updates},
	)
	return err
}

func (r *MetricRepositoryImpl) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *MetricRepositoryImpl) FindByDataSource(ctx context.Context, dataSourceID string) ([]models.Metric, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"data_source_id": dataSourceID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var metrics []models.Metric
	if err = cursor.All(ctx, &metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

func (r *MetricRepositoryImpl) FindByModule(ctx context.Context, module string) ([]models.Metric, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"module": module})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var metrics []models.Metric
	if err = cursor.All(ctx, &metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}
