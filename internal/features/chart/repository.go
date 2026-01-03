package chart

import (
	"context"

	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ChartRepository interface {
	Create(ctx context.Context, chart *Chart) error
	Get(ctx context.Context, id string) (*Chart, error)
	List(ctx context.Context) ([]Chart, error)
	Update(ctx context.Context, id string, chart *Chart) error
	Delete(ctx context.Context, id string) error
}

type ChartRepositoryImpl struct {
	Collection *mongo.Collection
}

func NewChartRepository(mongodb *database.MongodbDB) ChartRepository {
	return &ChartRepositoryImpl{
		Collection: mongodb.DB.Collection("charts"),
	}
}

func (r *ChartRepositoryImpl) Create(ctx context.Context, chart *Chart) error {
	chart.ID = primitive.NewObjectID()
	_, err := r.Collection.InsertOne(ctx, chart)
	return err
}

func (r *ChartRepositoryImpl) Get(ctx context.Context, id string) (*Chart, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var chart Chart
	err = r.Collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&chart)
	if err != nil {
		return nil, err
	}
	return &chart, nil
}

func (r *ChartRepositoryImpl) List(ctx context.Context) ([]Chart, error) {
	cursor, err := r.Collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var charts []Chart
	if err := cursor.All(ctx, &charts); err != nil {
		return nil, err
	}
	return charts, nil
}

func (r *ChartRepositoryImpl) Update(ctx context.Context, id string, chart *Chart) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": chart,
	}

	_, err = r.Collection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	return err
}

func (r *ChartRepositoryImpl) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.Collection.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
