package resource

import (
	"context"
	"fmt"
	"go-crm/internal/common/models"
	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ResourceRepository interface {
	FindAll(ctx context.Context) ([]Resource, error)
	FindByID(ctx context.Context, id string) (*Resource, error)
	FindByKey(ctx context.Context, key string) (*Resource, error)
	FindSidebarResources(ctx context.Context, product string, location string) ([]Resource, error)
	Create(ctx context.Context, resource *Resource) error
	Update(ctx context.Context, resource *Resource) error
	Delete(ctx context.Context, id string) error
}

type ResourceRepositoryImpl struct {
	collection *mongo.Collection
}

func NewResourceRepository(db *database.MongodbDB) ResourceRepository {
	return &ResourceRepositoryImpl{
		collection: db.DB.Collection("resources"),
	}
}

func (r *ResourceRepositoryImpl) FindAll(ctx context.Context) ([]Resource, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	cursor, err := r.collection.Find(ctx, bson.M{"tenant_id": oid})
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

func (r *ResourceRepositoryImpl) FindByID(ctx context.Context, id string) (*Resource, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	var resource Resource
	err = r.collection.FindOne(ctx, bson.M{"_id": id, "tenant_id": oid}).Decode(&resource)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

func (r *ResourceRepositoryImpl) FindByKey(ctx context.Context, key string) (*Resource, error) {
	var resource Resource
	err := r.collection.FindOne(ctx, bson.M{"key": key}).Decode(&resource)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

func (r *ResourceRepositoryImpl) Create(ctx context.Context, resource *Resource) error {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}
	resource.TenantID = oid

	_, err = r.collection.InsertOne(ctx, resource)
	return err
}

func (r *ResourceRepositoryImpl) Update(ctx context.Context, resource *Resource) error {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}

	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": resource.ID, "tenant_id": oid}, resource)
	return err
}

func (r *ResourceRepositoryImpl) Delete(ctx context.Context, id string) error {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": id, "tenant_id": oid})
	return err
}

func (r *ResourceRepositoryImpl) FindSidebarResources(ctx context.Context, product string, location string) ([]Resource, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"tenant_id":  oid,
		"ui.sidebar": true,
	}

	if product != "" {
		filter["product"] = product
	}

	if location != "" {
		filter["ui.location"] = location
	}

	// Sort by group (alphabetically) then by order within group
	opts := options.Find().SetSort(bson.D{
		{Key: "ui.group", Value: 1},
		{Key: "ui.order", Value: 1},
	})

	cursor, err := r.collection.Find(ctx, filter, opts)
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
