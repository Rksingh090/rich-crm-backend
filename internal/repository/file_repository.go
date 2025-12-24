package repository

import (
	"context"

	"go-crm/internal/database"
	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type FileRepository interface {
	Save(ctx context.Context, file *models.File) error
	Get(ctx context.Context, id string) (*models.File, error)
}

type FileRepositoryImpl struct {
	Collection *mongo.Collection
}

func NewFileRepository(mongodb *database.MongodbDB) FileRepository {
	return &FileRepositoryImpl{
		Collection: mongodb.DB.Collection("files"),
	}
}

func (r *FileRepositoryImpl) Save(ctx context.Context, file *models.File) error {
	if file.ID.IsZero() {
		file.ID = primitive.NewObjectID()
	}
	_, err := r.Collection.InsertOne(ctx, file)
	return err
}

func (r *FileRepositoryImpl) Get(ctx context.Context, id string) (*models.File, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var file models.File
	err = r.Collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&file)
	return &file, err
}
