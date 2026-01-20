package file

import (
	"context"
	"fmt"

	"go-crm/internal/common/models"
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
	db *database.MongodbDB
}

func NewFileRepository(mongodb *database.MongodbDB) FileRepository {
	return &FileRepositoryImpl{
		db: mongodb,
	}
}

func (r *FileRepositoryImpl) getCollection(ctx context.Context) (*mongo.Collection, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	return r.db.GetTenantDB(tenantID).Collection("files"), nil
}

func (r *FileRepositoryImpl) Save(ctx context.Context, file *File) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	if file.ID.IsZero() {
		file.ID = primitive.NewObjectID()
	}

	// Ensure tenant ID is set if available
	if tenantID, ok := ctx.Value(models.TenantIDKey).(string); ok && tenantID != "" {
		file.TenantID, _ = primitive.ObjectIDFromHex(tenantID)
	}

	_, err = coll.InsertOne(ctx, file)
	return err
}

func (r *FileRepositoryImpl) Get(ctx context.Context, id string) (*File, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var file File
	err = coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&file)
	return &file, err
}

func (r *FileRepositoryImpl) FindByRecord(ctx context.Context, moduleName, recordID string) ([]*File, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"module_name": moduleName,
		"record_id":   recordID,
	}
	cursor, err := coll.Find(ctx, filter)
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
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"is_shared": true}
	cursor, err := coll.Find(ctx, filter)
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
	coll, err := r.getCollection(ctx)
	if err != nil {
		return 0, err
	}

	filter := bson.M{
		"module_name": moduleName,
		"record_id":   recordID,
	}
	return coll.CountDocuments(ctx, filter)
}

func (r *FileRepositoryImpl) Delete(ctx context.Context, id string) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = coll.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
