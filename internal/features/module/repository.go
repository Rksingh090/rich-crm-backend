package module

import (
	"context"
	"fmt"

	"go-crm/internal/common/models"
	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ModuleRepository interface {
	Create(ctx context.Context, module *Module) error
	FindByName(ctx context.Context, name string) (*Module, error)
	List(ctx context.Context, product string) ([]Module, error)
	Update(ctx context.Context, module *Module) error
	Delete(ctx context.Context, name string) error
	FindUsingLookup(ctx context.Context, targetModule string) ([]Module, error)
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

func (r *ModuleRepositoryImpl) Create(ctx context.Context, module *Module) error {
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

func (r *ModuleRepositoryImpl) FindByName(ctx context.Context, name string) (*Module, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	var module Module
	// Lookup by Slug (name field in Entity struct seems to be internal name/slug)
	err = r.Collection.FindOne(ctx, bson.M{"name": name, "tenant_id": oid}).Decode(&module)
	if err != nil {
		return nil, err
	}
	return &module, nil
}

func (r *ModuleRepositoryImpl) List(ctx context.Context, product string) ([]Module, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"tenant_id": oid}
	if product != "" {
		filter["product"] = product
	}

	cursor, err := r.Collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var modules []Module
	if err = cursor.All(ctx, &modules); err != nil {
		return nil, err
	}
	return modules, nil
}

func (r *ModuleRepositoryImpl) Update(ctx context.Context, module *Module) error {
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

func (r *ModuleRepositoryImpl) Delete(ctx context.Context, name string) error {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}

	// 1. Delete associated records
	_, err = r.DB.Collection("entity_records").DeleteMany(ctx, bson.M{"entity": name, "tenant_id": oid})
	if err != nil {
		return fmt.Errorf("failed to delete module records: %w", err)
	}

	// 2. Delete module metadata
	_, err = r.Collection.DeleteOne(ctx, bson.M{"name": name, "tenant_id": oid})
	return err
}

func (r *ModuleRepositoryImpl) FindUsingLookup(ctx context.Context, targetModule string) ([]Module, error) {
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
		"tenant_id": oid,
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

	var modules []Module
	if err = cursor.All(ctx, &modules); err != nil {
		return nil, err
	}
	return modules, nil
}
