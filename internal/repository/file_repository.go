package repository

import (
	"context"

	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoFileRepository struct {
	Collection *mongo.Collection
}

func NewMongoFileRepository(db *mongo.Database) *MongoFileRepository {
	return &MongoFileRepository{
		Collection: db.Collection("files"),
	}
}

func (r *MongoFileRepository) Save(ctx context.Context, file *models.File) error {
	if file.ID.IsZero() {
		file.ID = primitive.NewObjectID()
	}
	_, err := r.Collection.InsertOne(ctx, file)
	return err
}

func (r *MongoFileRepository) Get(ctx context.Context, id string) (*models.File, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var file models.File
	err = r.Collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&file)
	return &file, err
}
