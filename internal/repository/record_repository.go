package repository

import (
	"context"
	"fmt"
	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RecordRepository interface {
	Create(ctx context.Context, moduleName string, data map[string]any) (any, error)
	Get(ctx context.Context, moduleName, id string) (map[string]any, error)
	List(ctx context.Context, moduleName string, filter map[string]any, limit, offset int64) ([]map[string]any, error)
	Update(ctx context.Context, moduleName, id string, data map[string]any) error
	Delete(ctx context.Context, moduleName, id string) error
}

type RecordRepositoryImpl struct {
	DB *mongo.Database
}

func NewRecordRepository(mongodb *database.MongodbDB) RecordRepository {
	return &RecordRepositoryImpl{
		DB: mongodb.DB,
	}
}

func (r *RecordRepositoryImpl) Create(ctx context.Context, moduleName string, data map[string]interface{}) (interface{}, error) {
	collectionName := fmt.Sprintf("module_%s", moduleName)
	result, err := r.DB.Collection(collectionName).InsertOne(ctx, data)
	if err != nil {
		return nil, err
	}
	return result.InsertedID, nil
}

func (r *RecordRepositoryImpl) Get(ctx context.Context, moduleName, id string) (map[string]interface{}, error) {
	collectionName := fmt.Sprintf("module_%s", moduleName)
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = r.DB.Collection(collectionName).FindOne(ctx, bson.M{"_id": oid}).Decode(&result)
	return result, err
}

func (r *RecordRepositoryImpl) List(ctx context.Context, moduleName string, filter map[string]any, limit, offset int64) ([]map[string]any, error) {
	collectionName := fmt.Sprintf("module_%s", moduleName)

	findOptions := options.Find()
	findOptions.SetLimit(limit)
	findOptions.SetSkip(offset)
	// Optional: Default sort by created_at desc
	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.DB.Collection(collectionName).Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []map[string]any
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *RecordRepositoryImpl) Update(ctx context.Context, moduleName, id string, data map[string]interface{}) error {
	collectionName := fmt.Sprintf("module_%s", moduleName)
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.DB.Collection(collectionName).UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": data})
	return err
}

func (r *RecordRepositoryImpl) Delete(ctx context.Context, moduleName, id string) error {
	collectionName := fmt.Sprintf("module_%s", moduleName)
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.DB.Collection(collectionName).DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
