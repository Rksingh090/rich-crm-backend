package resource

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ResourceRepository interface {
	Create(ctx context.Context, resource *Resource) error
	FindByID(ctx context.Context, resourceID string, tenantID *primitive.ObjectID) (*Resource, error)
	FindByResourceID(ctx context.Context, resourceID string) (*Resource, error)
	FindAll(ctx context.Context, filter ResourceFilter) ([]Resource, error)
	Update(ctx context.Context, id primitive.ObjectID, resource *Resource) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	SoftDelete(ctx context.Context, id primitive.ObjectID) error
	GetDefaults(ctx context.Context) ([]Resource, error)
	EnsureIndexes(ctx context.Context) error
}

type ResourceRepositoryImpl struct {
	collection *mongo.Collection
}

func NewResourceRepository(db *mongo.Database) ResourceRepository {
	return &ResourceRepositoryImpl{
		collection: db.Collection("resources"),
	}
}

func (r *ResourceRepositoryImpl) Create(ctx context.Context, resource *Resource) error {
	resource.CreatedAt = time.Now()
	resource.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, resource)
	return err
}

func (r *ResourceRepositoryImpl) FindByID(ctx context.Context, resourceID string, tenantID *primitive.ObjectID) (*Resource, error) {
	filter := bson.M{
		"resource_id": resourceID,
		"deleted_at":  nil,
	}

	// If tenantID is provided, match it or allow global/app-level resources
	if tenantID != nil {
		filter["$or"] = []bson.M{
			{"tenant_id": tenantID},
			{"tenant_id": nil}, // Global or app-level resources
		}
	} else {
		// If no tenantID, only return global/app-level resources
		filter["tenant_id"] = nil
	}

	var resource Resource
	err := r.collection.FindOne(ctx, filter).Decode(&resource)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("resource not found")
		}
		return nil, err
	}

	return &resource, nil
}

func (r *ResourceRepositoryImpl) FindByResourceID(ctx context.Context, resourceID string) (*Resource, error) {
	filter := bson.M{
		"resource_id": resourceID,
		"deleted_at":  nil,
	}

	var resource Resource
	err := r.collection.FindOne(ctx, filter).Decode(&resource)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("resource not found")
		}
		return nil, err
	}

	return &resource, nil
}

func (r *ResourceRepositoryImpl) FindAll(ctx context.Context, filter ResourceFilter) ([]Resource, error) {
	query := bson.M{
		"deleted_at": nil,
	}

	// Build query based on filter
	if filter.App != nil {
		query["app"] = *filter.App
	}

	if filter.Type != nil {
		query["type"] = *filter.Type
	}

	if filter.Scope != nil {
		query["scope"] = *filter.Scope
	}

	// Handle tenant filtering
	if filter.TenantID != nil {
		if filter.IncludeGlobal {
			// Include tenant-specific AND global/app-level resources
			query["$or"] = []bson.M{
				{"tenant_id": filter.TenantID},
				{"scope": ResourceScopeGlobal},
				{"scope": ResourceScopeApp},
			}
		} else {
			// Only tenant-specific resources
			query["tenant_id"] = filter.TenantID
		}
	} else {
		// No tenant specified - only global/app-level
		query["tenant_id"] = nil
	}

	opts := options.Find().SetSort(bson.D{
		{Key: "ui.group_order", Value: 1},
		{Key: "ui.order", Value: 1},
	})

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var resources []Resource
	if err := cursor.All(ctx, &resources); err != nil {
		return nil, err
	}

	return resources, nil
}

func (r *ResourceRepositoryImpl) Update(ctx context.Context, id primitive.ObjectID, resource *Resource) error {
	resource.UpdatedAt = time.Now()

	update := bson.M{
		"$set": resource,
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("resource not found")
	}

	return nil
}

func (r *ResourceRepositoryImpl) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("resource not found")
	}

	return nil
}

func (r *ResourceRepositoryImpl) SoftDelete(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"updated_at": now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("resource not found")
	}

	return nil
}

// GetDefaults retrieves default resources from control plane for tenant seeding
func (r *ResourceRepositoryImpl) GetDefaults(ctx context.Context) ([]Resource, error) {
	// Access control plane database for default resources
	db := r.collection.Database().Client().Database("control_plane")
	coll := db.Collection("default_resources")

	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var resources []Resource
	if err := cursor.All(ctx, &resources); err != nil {
		return nil, err
	}

	return resources, nil
}

// EnsureIndexes creates necessary indexes for the resources collection
func (r *ResourceRepositoryImpl) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "resource_id", Value: 1},
				{Key: "tenant_id", Value: 1},
			},
			Options: options.Index().SetName("idx_resource_tenant").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "app", Value: 1},
				{Key: "type", Value: 1},
			},
			Options: options.Index().SetName("idx_app_type"),
		},
		{
			Keys: bson.D{
				{Key: "scope", Value: 1},
			},
			Options: options.Index().SetName("idx_scope"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}
