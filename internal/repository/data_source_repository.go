package repository

import (
	"context"
	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DataSourceRepository interface {
	Create(ctx context.Context, dataSource *models.DataSource) error
	Get(ctx context.Context, id string) (*models.DataSource, error)
	List(ctx context.Context) ([]models.DataSource, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	FindByType(ctx context.Context, dsType string) ([]models.DataSource, error)
}

type DataSourceRepositoryImpl struct {
	collection *mongo.Collection
}

func NewDataSourceRepository(db *mongo.Database) DataSourceRepository {
	return &DataSourceRepositoryImpl{
		collection: db.Collection("data_sources"),
	}
}

func (r *DataSourceRepositoryImpl) Create(ctx context.Context, dataSource *models.DataSource) error {
	dataSource.ID = primitive.NewObjectID()
	_, err := r.collection.InsertOne(ctx, dataSource)
	return err
}

func (r *DataSourceRepositoryImpl) Get(ctx context.Context, id string) (*models.DataSource, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var dataSource models.DataSource
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&dataSource)
	if err != nil {
		return nil, err
	}

	return &dataSource, nil
}

func (r *DataSourceRepositoryImpl) List(ctx context.Context) ([]models.DataSource, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var dataSources []models.DataSource
	if err = cursor.All(ctx, &dataSources); err != nil {
		return nil, err
	}

	return dataSources, nil
}

func (r *DataSourceRepositoryImpl) Update(ctx context.Context, id string, updates map[string]interface{}) error {
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

func (r *DataSourceRepositoryImpl) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *DataSourceRepositoryImpl) FindByType(ctx context.Context, dsType string) ([]models.DataSource, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"type": dsType})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var dataSources []models.DataSource
	if err = cursor.All(ctx, &dataSources); err != nil {
		return nil, err
	}

	return dataSources, nil
}
