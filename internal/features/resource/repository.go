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
	Create(ctx context.Context, resource *models.Resource) error
	Update(ctx context.Context, resource *models.Resource) error
	Delete(ctx context.Context, id string, userID string) error
	FindAll(ctx context.Context) ([]models.Resource, error)
	FindByResourceID(ctx context.Context, resourceID string) (*models.Resource, error)
	FindSidebarResources(ctx context.Context, app string, location string) ([]models.Resource, error)
	GetDefaults(ctx context.Context) ([]models.Resource, error)
	EnsureIndexes(ctx context.Context) error
	EnsureGlobalIndexes(ctx context.Context) error
}

type ResourceRepositoryImpl struct {
	DB *database.MongodbDB
}

func NewResourceRepository(db *database.MongodbDB) ResourceRepository {
	return &ResourceRepositoryImpl{
		DB: db,
	}
}

func (r *ResourceRepositoryImpl) getCollection(ctx context.Context) *mongo.Collection {
	tenantID, _ := ctx.Value(models.TenantIDKey).(string)
	if tenantID != "" {
		return r.DB.GetTenantDB(tenantID).Collection("resources")
	}
	return r.DB.DB.Collection("resources")
}

func (r *ResourceRepositoryImpl) GetDefaults(ctx context.Context) ([]models.Resource, error) {
	db := r.DB.GetControlPlaneDB()
	coll := db.Collection("default_resources")
	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var resources []models.Resource
	if err := cursor.All(ctx, &resources); err != nil {
		return nil, err
	}
	return resources, nil
}

func (r *ResourceRepositoryImpl) FindAll(ctx context.Context) ([]models.Resource, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	// 1. Fetch Global Resources (Control Plane)
	globalColl := r.DB.GetControlPlaneDB().Collection("default_resources")
	globalFilter := bson.M{
		"scope":      "global",
		"deleted_at": bson.M{"$exists": false},
	}
	// App filter from context
	if app, ok := ctx.Value(models.AppIDKey).(string); ok && app != "" {
		globalFilter["app"] = app
	}

	globalCursor, err := globalColl.Find(ctx, globalFilter)
	if err != nil {
		return nil, err
	}
	defer globalCursor.Close(ctx)
	var globalResources []models.Resource
	if err := globalCursor.All(ctx, &globalResources); err != nil {
		return nil, err
	}

	// 2. Fetch Tenant Resources & Overrides (Tenant DB)
	tenantColl := r.getCollection(ctx)
	tenantFilter := bson.M{
		"$or": []bson.M{
			{"tenant_id": oid, "scope": "tenant"},   // Tenant-specific
			{"tenant_id": oid, "is_override": true}, // Overrides
		},
		"deleted_at": bson.M{"$exists": false},
	}
	if app, ok := ctx.Value(models.AppIDKey).(string); ok && app != "" {
		tenantFilter["app"] = app
	}

	tenantCursor, err := tenantColl.Find(ctx, tenantFilter)
	if err != nil {
		return nil, err
	}
	defer tenantCursor.Close(ctx)
	var tenantResources []models.Resource
	if err := tenantCursor.All(ctx, &tenantResources); err != nil {
		return nil, err
	}

	// 3. Merge: Tenant overrides/specifics replace globals
	resourceMap := make(map[string]models.Resource)

	// Base: Globals
	for _, res := range globalResources {
		resourceMap[res.ResourceID] = res
	}

	// Overlay: Tenants
	for _, res := range tenantResources {
		// If override, it replaces global.
		// If tenant specific (scope=tenant), it is added (or replaces if ID conflict, though unlikely for distinct resources)
		resourceMap[res.ResourceID] = res
	}

	// Convert map back to slice
	var result []models.Resource
	for _, res := range resourceMap {
		result = append(result, res)
	}

	// Optional: Sort by ID or name for consistency?
	sort.Slice(result, func(i, j int) bool {
		return result[i].ResourceID < result[j].ResourceID
	})

	return result, nil
}

func (r *ResourceRepositoryImpl) FindByResourceID(ctx context.Context, resourceID string) (*models.Resource, error) {
	tenantID, _ := ctx.Value(models.TenantIDKey).(string)
	var oid primitive.ObjectID
	if tenantID != "" {
		oid, _ = primitive.ObjectIDFromHex(tenantID)
	}

	// 1. Try to find tenant-specific or override in Tenant DB
	if !oid.IsZero() {
		filter := bson.M{
			"resource_id": resourceID,
			"tenant_id":   oid,
			"deleted_at":  bson.M{"$exists": false},
		}
		coll := r.getCollection(ctx)
		var res models.Resource
		err := coll.FindOne(ctx, filter).Decode(&res)
		if err == nil {
			return &res, nil
		}
	}

	// 2. Fallback to global in Control Plane DB
	filter := bson.M{
		"resource_id": resourceID,
		// "scope":       "global", // Not strictly needed if default_resources contains only globals
		"deleted_at": bson.M{"$exists": false},
	}
	db := r.DB.GetControlPlaneDB()
	coll := db.Collection("default_resources")
	var res models.Resource
	err := coll.FindOne(ctx, filter).Decode(&res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *ResourceRepositoryImpl) Create(ctx context.Context, resource *models.Resource) error {
	if resource.Scope == "global" {
		db := r.DB.GetControlPlaneDB()
		coll := db.Collection("default_resources")
		// Ensure timestamps
		if resource.CreatedAt.IsZero() {
			resource.CreatedAt = time.Now()
		}
		if resource.UpdatedAt.IsZero() {
			resource.UpdatedAt = time.Now()
		}
		_, err := coll.InsertOne(ctx, resource)
		return err
	}

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

	coll := r.getCollection(ctx)
	// Ensure timestamps
	if resource.CreatedAt.IsZero() {
		resource.CreatedAt = time.Now()
	}
	if resource.UpdatedAt.IsZero() {
		resource.UpdatedAt = time.Now()
	}
	_, err := coll.InsertOne(ctx, resource)
	return err
}

func (r *ResourceRepositoryImpl) Update(ctx context.Context, resource *models.Resource) error {
	if resource.Scope == "global" {
		db := r.DB.GetControlPlaneDB()
		coll := db.Collection("default_resources")

		resource.UpdatedAt = time.Now()
		filter := bson.M{"_id": resource.ID}
		update := bson.M{"$set": resource}

		_, err := coll.UpdateOne(ctx, filter, update)
		return err
	}

	filter := bson.M{"_id": resource.ID}

	// For non-global resources, strictly enforce tenant_id
	if resource.Scope != "global" {
		tenantID, ok := ctx.Value(models.TenantIDKey).(string)
		if !ok || tenantID == "" {
			return fmt.Errorf("organization context missing")
		}
		oid, _ := primitive.ObjectIDFromHex(tenantID)
		filter["tenant_id"] = oid
	}

	resource.UpdatedAt = time.Now()
	coll := r.getCollection(ctx)
	_, err := coll.ReplaceOne(ctx, filter, resource)
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
	coll := r.getCollection(ctx)
	_, err = coll.UpdateOne(ctx, filter, update)
	return err
}

func (r *ResourceRepositoryImpl) FindSidebarResources(ctx context.Context, app string, location string) ([]models.Resource, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	// 1. Fetch Global Resources (Control Plane)
	globalColl := r.DB.GetControlPlaneDB().Collection("default_resources")
	globalFilter := bson.M{
		"ui.sidebar": true,
		"deleted_at": bson.M{"$exists": false},
	}
	if app != "" {
		globalFilter["app"] = app
	}
	if location != "" {
		globalFilter["ui.location"] = location
	}

	globalCursor, err := globalColl.Find(ctx, globalFilter)
	if err != nil {
		return nil, err
	}
	defer globalCursor.Close(ctx)
	var globalResources []models.Resource
	if err := globalCursor.All(ctx, &globalResources); err != nil {
		return nil, err
	}

	// 2. Fetch Tenant Resources & Overrides (Tenant DB)
	tenantColl := r.getCollection(ctx)
	tenantFilter := bson.M{
		"$or": []bson.M{
			{"tenant_id": oid, "scope": "tenant"},   // Tenant-specific
			{"tenant_id": oid, "is_override": true}, // Overrides
		},
		"ui.sidebar": true,
		"deleted_at": bson.M{"$exists": false},
	}
	if app != "" {
		tenantFilter["app"] = app
	}
	if location != "" {
		tenantFilter["ui.location"] = location
	}

	tenantCursor, err := tenantColl.Find(ctx, tenantFilter)
	if err != nil {
		return nil, err
	}
	defer tenantCursor.Close(ctx)
	var tenantResources []models.Resource
	if err := tenantCursor.All(ctx, &tenantResources); err != nil {
		return nil, err
	}

	// 3. Merge: Tenant overrides replace globals
	resourceMap := make(map[string]models.Resource)

	// Base: Globals
	for _, res := range globalResources {
		resourceMap[res.ResourceID] = res
	}

	// Overlay: Tenants
	for _, res := range tenantResources {
		resourceMap[res.ResourceID] = res
	}

	// Convert to slice
	var result []models.Resource
	for _, res := range resourceMap {
		result = append(result, res)
	}

	// 4. Sort
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
				{Key: "resource_id", Value: 1},
				{Key: "tenant_id", Value: 1},
			},
			Options: options.Index().SetName("idx_resource_tenant").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "ui.sidebar", Value: 1},
				{Key: "tenant_id", Value: 1},
			},
			Options: options.Index().SetName("idx_sidebar_tenant"),
		},
	}

	coll := r.getCollection(ctx)
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *ResourceRepositoryImpl) EnsureGlobalIndexes(ctx context.Context) error {
	db := r.DB.GetControlPlaneDB()
	coll := db.Collection("default_resources")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "resource_id", Value: 1},
			},
			Options: options.Index().SetName("idx_template_resource_id").SetUnique(true),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
