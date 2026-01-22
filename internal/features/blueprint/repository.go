package blueprint

import (
	"context"
	"fmt"
	"time"

	"go-crm/internal/common/models"
	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Repository interface {
	Create(ctx context.Context, blueprint *Blueprint) error
	Update(ctx context.Context, blueprint *Blueprint) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*Blueprint, error)
	List(ctx context.Context, filter BlueprintFilter) ([]Blueprint, error)
	FindActiveByModule(ctx context.Context, module string) (*Blueprint, error)
}

type RepositoryImpl struct {
	DB *database.MongodbDB
}

func NewRepository(db *database.MongodbDB) Repository {
	return &RepositoryImpl{
		DB: db,
	}
}

func (r *RepositoryImpl) getCollection(ctx context.Context) (*mongo.Collection, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	db := r.DB.GetTenantDB(tenantID)
	return db.Collection("blueprints"), nil
}

func (r *RepositoryImpl) Create(ctx context.Context, blueprint *Blueprint) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	oid, _ := primitive.ObjectIDFromHex(ctx.Value(models.TenantIDKey).(string))
	blueprint.TenantID = oid
	blueprint.CreatedAt = time.Now()
	blueprint.UpdatedAt = time.Now()

	result, err := coll.InsertOne(ctx, blueprint)
	if err != nil {
		return err
	}
	blueprint.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *RepositoryImpl) Update(ctx context.Context, blueprint *Blueprint) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	blueprint.UpdatedAt = time.Now()
	filter := bson.M{"_id": blueprint.ID}
	update := bson.M{"$set": blueprint}

	_, err = coll.UpdateOne(ctx, filter, update)
	return err
}

func (r *RepositoryImpl) Delete(ctx context.Context, id string) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	// Soft delete: set deleted=true
	update := bson.M{
		"$set": bson.M{
			"__deleted":  true,
			"updated_at": time.Now(),
		},
	}

	_, err = coll.UpdateOne(ctx, bson.M{"_id": oid}, update)
	return err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id string) (*Blueprint, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var blueprint Blueprint
	// Exclude soft-deleted records
	err = coll.FindOne(ctx, bson.M{
		"_id": oid,
		"$or": []bson.M{
			{"__deleted": bson.M{"$exists": false}},
			{"__deleted": false},
		},
	}).Decode(&blueprint)
	if err != nil {
		return nil, err
	}
	return &blueprint, nil
}

func (r *RepositoryImpl) List(ctx context.Context, filter BlueprintFilter) ([]Blueprint, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	// Exclude soft-deleted records
	query := bson.M{
		"$or": []bson.M{
			{"__deleted": bson.M{"$exists": false}},
			{"__deleted": false},
		},
	}
	if filter.Module != "" {
		query["module"] = filter.Module
	}
	if filter.Search != "" {
		query["$or"] = []bson.M{
			{"name": bson.M{"$regex": filter.Search, "$options": "i"}},
			{"module": bson.M{"$regex": filter.Search, "$options": "i"}},
		}
	}

	cursor, err := coll.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var blueprints []Blueprint
	if err = cursor.All(ctx, &blueprints); err != nil {
		return nil, err
	}
	return blueprints, nil
}

func (r *RepositoryImpl) FindActiveByModule(ctx context.Context, module string) (*Blueprint, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	// Assuming only one active blueprint per module for now, or per target field?
	// For now, let's assume we want the active one for a given module.
	// We might need to filter by TargetField as well if we allow multiple per module on different fields.
	// But usually one main blueprint per module is a good start.

	var blueprint Blueprint
	// Exclude soft-deleted records
	err = coll.FindOne(ctx, bson.M{
		"module": module,
		"active": true,
		"$or": []bson.M{
			{"__deleted": bson.M{"$exists": false}},
			{"__deleted": false},
		},
	}).Decode(&blueprint)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &blueprint, nil
}
