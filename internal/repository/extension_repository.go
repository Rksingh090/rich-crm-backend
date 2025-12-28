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

type ExtensionRepository interface {
	Create(ctx context.Context, ext *models.Extension) error
	GetByID(ctx context.Context, id string) (*models.Extension, error)
	List(ctx context.Context, onlyInstalled bool) ([]models.Extension, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
}

type ExtensionRepositoryImpl struct {
	Collection *mongo.Collection
}

func NewExtensionRepository(mongodb *database.MongodbDB) ExtensionRepository {
	return &ExtensionRepositoryImpl{
		Collection: mongodb.DB.Collection("extensions"),
	}
}

func (r *ExtensionRepositoryImpl) Create(ctx context.Context, ext *models.Extension) error {
	ext.ID = primitive.NewObjectID()
	ext.CreatedAt = time.Now()
	ext.UpdatedAt = time.Now()
	_, err := r.Collection.InsertOne(ctx, ext)
	return err
}

func (r *ExtensionRepositoryImpl) GetByID(ctx context.Context, id string) (*models.Extension, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var ext models.Extension
	err = r.Collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&ext)
	if err != nil {
		return nil, err
	}
	return &ext, nil
}

func (r *ExtensionRepositoryImpl) List(ctx context.Context, onlyInstalled bool) ([]models.Extension, error) {
	filter := bson.M{}
	if onlyInstalled {
		filter["installed"] = true
	}

	cursor, err := r.Collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var extensions []models.Extension
	if err = cursor.All(ctx, &extensions); err != nil {
		return nil, err
	}
	return extensions, nil
}

func (r *ExtensionRepositoryImpl) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	updates["updated_at"] = time.Now()
	_, err = r.Collection.UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": updates})
	return err
}

func (r *ExtensionRepositoryImpl) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.Collection.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
