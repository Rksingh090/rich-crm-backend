package resource

import (
	"context"
	"fmt"
	"go-crm/internal/common/models"
	"go-crm/internal/database"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ResourceRepository interface {
	FindAll(ctx context.Context) ([]Resource, error)
	FindByID(ctx context.Context, id string) (*Resource, error)
	FindByResourceID(ctx context.Context, resourceID string) (*Resource, error)
	FindByKey(ctx context.Context, key string) (*Resource, error)
	FindSidebarResources(ctx context.Context, product string, location string) ([]Resource, error)
	Create(ctx context.Context, resource *Resource) error
	Update(ctx context.Context, resource *Resource) error
	Delete(ctx context.Context, id string, userID string) error

	// Global and tenant-specific methods
	FindGlobalResources(ctx context.Context) ([]Resource, error)
	FindTenantResources(ctx context.Context) ([]Resource, error)
	FindTenantOverrides(ctx context.Context) ([]Resource, error)
	FindMergedResources(ctx context.Context) ([]Resource, error)
	EnsureIndexes(ctx context.Context) error
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

	// Build filter to get both global resources and tenant-specific resources
	filter := bson.M{
		"$or": []bson.M{
			{"scope": "global"},                     // Global resources (no tenant_id)
			{"tenant_id": oid, "scope": "tenant"},   // Tenant-specific resources
			{"tenant_id": oid, "is_override": true}, // Tenant overrides
		},
		"deleted_at": bson.M{"$exists": false},
	}

	// Read product from context (set by middleware/controller)
	if product, ok := ctx.Value("product").(string); ok && product != "" {
		filter["product"] = product
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var resources []Resource
	if err := cursor.All(ctx, &resources); err != nil {
		return nil, err
	}

	// Merge resources: tenant overrides should replace global resources with same ResourceID
	resourceMap := make(map[string]Resource)

	for _, res := range resources {
		existing, exists := resourceMap[res.ResourceID]
		if !exists {
			// First time seeing this resource
			resourceMap[res.ResourceID] = res
		} else {
			// Resource already exists, check if current one should override
			// Priority: tenant override > tenant-specific > global
			if res.IsOverride {
				// Override always wins
				resourceMap[res.ResourceID] = res
			} else if res.Scope == "tenant" && existing.Scope == "global" {
				// Tenant-specific wins over global
				resourceMap[res.ResourceID] = res
			}
			// Otherwise keep existing
		}
	}

	// Convert map back to slice
	var result []Resource
	for _, res := range resourceMap {
		result = append(result, res)
	}

	return result, nil
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

	rid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid resource id: %v", err)
	}

	filter := bson.M{
		"_id":        rid,
		"tenant_id":  oid,
		"deleted_at": bson.M{"$exists": false},
	}
	var resource Resource
	err = r.collection.FindOne(ctx, filter).Decode(&resource)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

func (r *ResourceRepositoryImpl) FindByResourceID(ctx context.Context, resourceID string) (*Resource, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"resource":   resourceID,
		"tenant_id":  oid,
		"deleted_at": bson.M{"$exists": false},
	}
	var resource Resource
	err = r.collection.FindOne(ctx, filter).Decode(&resource)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

func (r *ResourceRepositoryImpl) FindByKey(ctx context.Context, key string) (*Resource, error) {
	filter := bson.M{
		"key":        key,
		"deleted_at": bson.M{"$exists": false},
	}
	var resource Resource
	err := r.collection.FindOne(ctx, filter).Decode(&resource)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

func (r *ResourceRepositoryImpl) Create(ctx context.Context, resource *Resource) error {
	// For tenant-scoped resources, require tenant context
	if resource.Scope == "tenant" && !resource.IsOverride {
		tenantID, ok := ctx.Value(models.TenantIDKey).(string)
		if !ok || tenantID == "" {
			return fmt.Errorf("organization context missing")
		}
		oid, err := primitive.ObjectIDFromHex(tenantID)
		if err != nil {
			return err
		}
		resource.TenantID = oid
	}
	// For global resources, TenantID should be empty
	// For overrides, TenantID is already set

	_, err := r.collection.InsertOne(ctx, resource)
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

func (r *ResourceRepositoryImpl) Delete(ctx context.Context, id string, userID string) error {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}

	rid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid resource id: %v", err)
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"deleted_by": userID,
		},
	}
	filter := bson.M{"_id": rid, "tenant_id": oid}
	_, err = r.collection.UpdateOne(ctx, filter, update)
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

	// Build filter to get both global resources and tenant-specific resources
	filter := bson.M{
		"$or": []bson.M{
			{"scope": "global"},                     // Global resources (no tenant_id)
			{"tenant_id": oid, "scope": "tenant"},   // Tenant-specific resources
			{"tenant_id": oid, "is_override": true}, // Tenant overrides
		},
		"ui.sidebar": true,
		"deleted_at": bson.M{"$exists": false},
	}

	if product != "" {
		filter["product"] = product
	}

	if location != "" {
		filter["ui.location"] = location
	}

	// Sort by group order, then group name, then by item order
	opts := options.Find().SetSort(bson.D{
		{Key: "ui.group_order", Value: 1},
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

	// Merge resources: tenant overrides should replace global resources with same ResourceID
	resourceMap := make(map[string]Resource)

	for _, res := range resources {
		existing, exists := resourceMap[res.ResourceID]
		if !exists {
			// First time seeing this resource
			resourceMap[res.ResourceID] = res
		} else {
			// Resource already exists, check if current one should override
			// Priority: tenant override > tenant-specific > global
			if res.IsOverride {
				// Override always wins
				resourceMap[res.ResourceID] = res
			} else if res.Scope == "tenant" && existing.Scope == "global" {
				// Tenant-specific wins over global
				resourceMap[res.ResourceID] = res
			}
			// Otherwise keep existing
		}
	}

	// Convert map back to slice
	var result []Resource
	for _, res := range resourceMap {
		result = append(result, res)
	}

	// Sort the result to maintain proper order
	// Sort by group order, then group name, then by item order
	sort.Slice(result, func(i, j int) bool {
		if result[i].UI.GroupOrder != result[j].UI.GroupOrder {
			return result[i].UI.GroupOrder < result[j].UI.GroupOrder
		}
		if result[i].UI.Group != result[j].UI.Group {
			return result[i].UI.Group < result[j].UI.Group
		}
		return result[i].UI.Order < result[j].UI.Order
	})

	return result, nil
}

// FindGlobalResources returns all global resources (scope="global")
func (r *ResourceRepositoryImpl) FindGlobalResources(ctx context.Context) ([]Resource, error) {
	filter := bson.M{
		"scope":      "global",
		"deleted_at": bson.M{"$exists": false},
	}

	cursor, err := r.collection.Find(ctx, filter)
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

// FindTenantResources returns all tenant-specific resources (scope="tenant", not overrides)
func (r *ResourceRepositoryImpl) FindTenantResources(ctx context.Context) ([]Resource, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"tenant_id":   oid,
		"scope":       "tenant",
		"is_override": false,
		"deleted_at":  bson.M{"$exists": false},
	}

	cursor, err := r.collection.Find(ctx, filter)
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

// FindTenantOverrides returns all tenant overrides for global resources
func (r *ResourceRepositoryImpl) FindTenantOverrides(ctx context.Context) ([]Resource, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"tenant_id":   oid,
		"is_override": true,
		"deleted_at":  bson.M{"$exists": false},
	}

	cursor, err := r.collection.Find(ctx, filter)
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

// FindMergedResources returns global resources merged with tenant overrides and tenant-specific resources
func (r *ResourceRepositoryImpl) FindMergedResources(ctx context.Context) ([]Resource, error) {
	// Get global resources
	globalResources, err := r.FindGlobalResources(ctx)
	if err != nil {
		return nil, err
	}

	// Get tenant overrides
	overrides, err := r.FindTenantOverrides(ctx)
	if err != nil {
		return nil, err
	}

	// Get tenant-specific resources
	tenantResources, err := r.FindTenantResources(ctx)
	if err != nil {
		return nil, err
	}

	// Create a map to merge global resources with overrides
	resourceMap := make(map[string]Resource)

	// Add global resources
	for _, res := range globalResources {
		resourceMap[res.ResourceID] = res
	}

	// Override with tenant overrides
	for _, override := range overrides {
		resourceMap[override.ResourceID] = override
	}

	// Add tenant-specific resources
	for _, res := range tenantResources {
		resourceMap[res.ResourceID] = res
	}

	// Convert map to slice
	var result []Resource
	for _, res := range resourceMap {
		result = append(result, res)
	}

	return result, nil
}

func (r *ResourceRepositoryImpl) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "tenant_id", Value: 1},
				{Key: "scope", Value: 1},
				{Key: "is_override", Value: 1},
			},
			Options: options.Index().SetName("idx_tenant_scope_override"),
		},
		{
			Keys: bson.D{
				{Key: "resource", Value: 1},
				{Key: "tenant_id", Value: 1},
			},
			Options: options.Index().SetName("idx_resource_tenant"),
		},
		{
			Keys: bson.D{
				{Key: "ui.sidebar", Value: 1},
				{Key: "tenant_id", Value: 1},
			},
			Options: options.Index().SetName("idx_sidebar_tenant"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}
