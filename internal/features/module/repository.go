package module

import (
	"context"
	"fmt"
	"time"

	"go-crm/internal/common/models"
	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ModuleRepository interface {
	Create(ctx context.Context, module *models.Entity) error
	FindByName(ctx context.Context, name string) (*models.Entity, error)
	List(ctx context.Context) ([]models.Entity, error)
	Update(ctx context.Context, module *models.Entity) error
	Delete(ctx context.Context, name string, userID string) error
	FindUsingLookup(ctx context.Context, targetModule string) ([]models.Entity, error)
	EnsureIndexes(ctx context.Context) error
}

type ModuleRepositoryImpl struct {
	Collection *mongo.Collection
	DB         *mongo.Database
}

func NewModuleRepository(mongodb *database.MongodbDB) ModuleRepository {
	return &ModuleRepositoryImpl{
		Collection: mongodb.DB.Collection("entities"),
		DB:         mongodb.DB,
	}
}

func (r *ModuleRepositoryImpl) Create(ctx context.Context, module *models.Entity) error {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		// Allow creation of global modules without tenant context
		if module.Scope == "global" {
			module.TenantID = primitive.NilObjectID
			_, err := r.Collection.InsertOne(ctx, module)
			return err
		}
		return fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}
	module.TenantID = oid

	// Default to tenant scope if not set (which is standard for this path)
	if module.Scope == "" {
		module.Scope = "tenant"
	}

	_, err = r.Collection.InsertOne(ctx, module)
	return err
}

func (r *ModuleRepositoryImpl) FindByName(ctx context.Context, name string) (*models.Entity, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	// 1. Try to find tenant-specific or override
	filter := bson.M{
		"name":       name,
		"tenant_id":  oid,
		"deleted_at": bson.M{"$exists": false},
	}
	var module models.Entity
	err = r.Collection.FindOne(ctx, filter).Decode(&module)
	if err == nil {
		return &module, nil
	}

	// 2. Fallback to global
	// Note: Global entities must implement "tenant isolation" by not being in tenant_id query
	// but we must explicitly look for scope=global and NO tenant_id (or ignore tenant_id for global?)
	// Common model: Global entities have empty tenant_id or special tenant_id.
	// We'll assume empty/null tenant_id means global OR scope="global" is sufficient differentiator.
	// But `tenant_id` index is unique?
	// Existing unique index is {name, tenant_id}.
	// If global entity has null tenant_id, then uniqueness works for globals.
	// So we search for {name: name, scope: "global"}
	globalFilter := bson.M{
		"name":       name,
		"scope":      "global",
		"deleted_at": bson.M{"$exists": false},
	}
	err = r.Collection.FindOne(ctx, globalFilter).Decode(&module)
	if err != nil {
		return nil, err
	}
	return &module, nil
}

func (r *ModuleRepositoryImpl) List(ctx context.Context) ([]models.Entity, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	// Fetch Global Entities
	globalFilter := bson.M{
		"scope":      "global",
		"deleted_at": bson.M{"$exists": false},
	}
	// Filter by product if set
	if product, ok := ctx.Value("product").(string); ok && product != "" {
		globalFilter["product"] = product
	}

	globalCursor, err := r.Collection.Find(ctx, globalFilter)
	if err != nil {
		return nil, err
	}
	defer globalCursor.Close(ctx)
	var globalModules []models.Entity
	if err = globalCursor.All(ctx, &globalModules); err != nil {
		return nil, err
	}

	// Fetch Tenant Entities (and Overrides)
	tenantFilter := bson.M{
		"tenant_id":  oid,
		"deleted_at": bson.M{"$exists": false},
	}
	if product, ok := ctx.Value("product").(string); ok && product != "" {
		tenantFilter["product"] = product
	}

	tenantCursor, err := r.Collection.Find(ctx, tenantFilter)
	if err != nil {
		return nil, err
	}
	defer tenantCursor.Close(ctx)
	var tenantModules []models.Entity
	if err = tenantCursor.All(ctx, &tenantModules); err != nil {
		return nil, err
	}

	// Merge: map by Name
	moduleMap := make(map[string]models.Entity)

	// Add globals first
	for _, m := range globalModules {
		moduleMap[m.Name] = m
	}

	// Override with tenant modules
	for _, m := range tenantModules {
		moduleMap[m.Name] = m
	}

	// Convert back to slice
	modules := make([]models.Entity, 0, len(moduleMap))
	for _, m := range moduleMap {
		modules = append(modules, m)
	}

	return modules, nil
}

func (r *ModuleRepositoryImpl) Update(ctx context.Context, module *models.Entity) error {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}

	filter := bson.M{"name": module.Name, "tenant_id": oid}
	update := bson.M{"$set": module}
	_, err = r.Collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *ModuleRepositoryImpl) Delete(ctx context.Context, name string, userID string) error {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}

	// Soft delete: set deleted_at and deleted_by
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"deleted_by": userID,
		},
	}
	filter := bson.M{"name": name, "tenant_id": oid}
	_, err = r.Collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *ModuleRepositoryImpl) FindUsingLookup(ctx context.Context, targetModule string) ([]models.Entity, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	// Find Tenant modules
	tenantFilter := bson.M{
		"tenant_id":  oid,
		"deleted_at": bson.M{"$exists": false},
		"fields": bson.M{
			"$elemMatch": bson.M{
				"type":          "lookup",
				"lookup.module": targetModule,
			},
		},
	}

	tenantCursor, err := r.Collection.Find(ctx, tenantFilter)
	if err != nil {
		return nil, err
	}
	defer tenantCursor.Close(ctx)
	var tenantModules []models.Entity
	if err = tenantCursor.All(ctx, &tenantModules); err != nil {
		return nil, err
	}

	// Find Global modules
	globalFilter := bson.M{
		"scope":      "global",
		"deleted_at": bson.M{"$exists": false},
		"fields": bson.M{
			"$elemMatch": bson.M{
				"type":          "lookup",
				"lookup.module": targetModule,
			},
		},
	}
	globalCursor, err := r.Collection.Find(ctx, globalFilter)
	if err != nil {
		return nil, err
	}
	defer globalCursor.Close(ctx)
	var globalModules []models.Entity
	if err = globalCursor.All(ctx, &globalModules); err != nil {
		return nil, err
	}

	// Merge
	moduleMap := make(map[string]models.Entity)
	for _, m := range globalModules {
		moduleMap[m.Name] = m
	}
	for _, m := range tenantModules {
		moduleMap[m.Name] = m
	}

	var modules []models.Entity
	for _, m := range moduleMap {
		modules = append(modules, m)
	}
	return modules, nil
}

func (r *ModuleRepositoryImpl) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "name", Value: 1},
				{Key: "tenant_id", Value: 1},
			},
			Options: options.Index().SetName("idx_name_tenant").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "fields.lookup.module", Value: 1},
				{Key: "tenant_id", Value: 1},
			},
			Options: options.Index().SetName("idx_lookup_refs"),
		},
		{
			Keys: bson.D{
				{Key: "scope", Value: 1},
			},
			Options: options.Index().SetName("idx_scope"),
		},
	}
	// Note: sparse or partial index could be used for lookup refs, but standard is fine
	_, err := r.Collection.Indexes().CreateMany(ctx, indexes)
	return err
}
