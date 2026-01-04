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
		return fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}
	module.TenantID = oid

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

	filter := bson.M{
		"name":       name,
		"tenant_id":  oid,
		"deleted_at": bson.M{"$exists": false},
	}
	var module models.Entity
	err = r.Collection.FindOne(ctx, filter).Decode(&module)
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

	filter := bson.M{
		"tenant_id":  oid,
		"deleted_at": bson.M{"$exists": false},
	}

	// Read product from context (set by middleware)
	if product, ok := ctx.Value("product").(string); ok && product != "" {
		filter["product"] = product
	}

	cursor, err := r.Collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var modules []models.Entity
	if err = cursor.All(ctx, &modules); err != nil {
		return nil, err
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

	// Find modules that have at least one field where field.lookup.module == targetModule
	filter := bson.M{
		"tenant_id":  oid,
		"deleted_at": bson.M{"$exists": false},
		"fields": bson.M{
			"$elemMatch": bson.M{
				"type":          "lookup",
				"lookup.module": targetModule,
			},
		},
	}

	cursor, err := r.Collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var modules []models.Entity
	if err = cursor.All(ctx, &modules); err != nil {
		return nil, err
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
	}
	// Note: sparse or partial index could be used for lookup refs, but standard is fine
	_, err := r.Collection.Indexes().CreateMany(ctx, indexes)
	return err
}
