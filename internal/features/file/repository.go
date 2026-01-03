package file

import (
	"context"

	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type FileRepository interface {
	Save(ctx context.Context, file *File) error
	Get(ctx context.Context, id string) (*File, error)
	FindByRecord(ctx context.Context, moduleName, recordID string) ([]*File, error)
	FindShared(ctx context.Context) ([]*File, error)
	CountByRecord(ctx context.Context, moduleName, recordID string) (int64, error)
	Delete(ctx context.Context, id string) error
}

type FileRepositoryImpl struct {
	Collection *mongo.Collection
}

func NewFileRepository(mongodb *database.MongodbDB) FileRepository {
	return &FileRepositoryImpl{
		Collection: mongodb.DB.Collection("files"),
	}
}

func (r *FileRepositoryImpl) Save(ctx context.Context, file *File) error {
	if file.ID.IsZero() {
		file.ID = primitive.NewObjectID()
	}
	_, err := r.Collection.InsertOne(ctx, file)
	return err
}

func (r *FileRepositoryImpl) Get(ctx context.Context, id string) (*File, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var file File
	err = r.Collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&file)
	return &file, err
}

func (r *FileRepositoryImpl) FindByRecord(ctx context.Context, moduleName, recordID string) ([]*File, error) {
	filter := bson.M{
		"module_name": moduleName,
		"record_id":   recordID,
	}
	cursor, err := r.Collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var files []*File
	if err := cursor.All(ctx, &files); err != nil {
		return nil, err
	}
	return files, nil
}

func (r *FileRepositoryImpl) FindShared(ctx context.Context) ([]*File, error) {
	filter := bson.M{"is_shared": true}
	cursor, err := r.Collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var files []*File
	if err := cursor.All(ctx, &files); err != nil {
		return nil, err
	}
	return files, nil
}

func (r *FileRepositoryImpl) CountByRecord(ctx context.Context, moduleName, recordID string) (int64, error) {
	filter := bson.M{
		"module_name": moduleName,
		"record_id":   recordID,
	}
	return r.Collection.CountDocuments(ctx, filter)
}

func (r *FileRepositoryImpl) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.Collection.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
