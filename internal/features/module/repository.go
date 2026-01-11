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
	EnsureGlobalIndexes(ctx context.Context) error
	GetDefaults(ctx context.Context) ([]models.Entity, error)
}

type ModuleRepositoryImpl struct {
	DB *database.MongodbDB
}

func NewModuleRepository(mongodb *database.MongodbDB) ModuleRepository {
	return &ModuleRepositoryImpl{
		DB: mongodb,
	}
}

// helper
func (r *ModuleRepositoryImpl) getCollection(ctx context.Context) (*mongo.Collection, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	db := r.DB.GetTenantDB(tenantID)
	return db.Collection("modules"), nil // Renaming 'entities' to 'modules' to be consistent or stick to 'entities'?
	// Previous code used "entities". Let's stick to "entities" for consistency unless user asked.
	// User said "default resources will be copied".
	// Let's use "modules" as collection name in Tenant DB to match feature name?
	// Or stay with "entities". Current codebase uses "entities". I'll stick to "entities".
}

func (r *ModuleRepositoryImpl) GetDefaults(ctx context.Context) ([]models.Entity, error) {
	// Read from Control Plane
	db := r.DB.GetControlPlaneDB()
	coll := db.Collection("default_modules") // Explicit separate collection for templates

	// If default_modules is empty, maybe fallback to "entities" with scope=global?
	// User said "control_plane_db where ... default resources ... will live".
	// I'll assume a new collection "default_modules".

	cursor, err := coll.Find(ctx, bson.M{})
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

func (r *ModuleRepositoryImpl) Create(ctx context.Context, module *models.Entity) error {
	if module.Scope == "global" {
		db := r.DB.GetControlPlaneDB()
		coll := db.Collection("default_modules")
		_, err := coll.InsertOne(ctx, module)
		return err
	}

	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	oid, _ := primitive.ObjectIDFromHex(ctx.Value(models.TenantIDKey).(string))
	module.TenantID = oid

	// Scope is always tenant in Tenant DB
	module.Scope = "tenant"

	_, err = coll.InsertOne(ctx, module)
	return err
}

func (r *ModuleRepositoryImpl) FindByName(ctx context.Context, name string) (*models.Entity, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	var module models.Entity
	err = coll.FindOne(ctx, bson.M{"name": name, "deleted_at": bson.M{"$exists": false}}).Decode(&module)
	if err == nil {
		return &module, nil
	}
	if err != mongo.ErrNoDocuments {
		return nil, err
	}

	// Fallback to Global
	db := r.DB.GetControlPlaneDB()
	defaultColl := db.Collection("default_modules")
	if err := defaultColl.FindOne(ctx, bson.M{"name": name, "deleted_at": bson.M{"$exists": false}}).Decode(&module); err != nil {
		return nil, err
	}

	// Ensure we preserve the scope as global so UI/Backend knows it's a default
	return &module, nil
}

func (r *ModuleRepositoryImpl) List(ctx context.Context) ([]models.Entity, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	// Simple List in Tenant DB
	filter := bson.M{"deleted_at": bson.M{"$exists": false}}
	if product, ok := ctx.Value(models.AppIDKey).(string); ok && product != "" {
		filter["app"] = product
	}

	cursor, err := coll.Find(ctx, filter)
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
	if module.Scope == "global" {
		db := r.DB.GetControlPlaneDB()
		coll := db.Collection("default_modules")

		filter := bson.M{"name": module.Name}
		update := bson.M{"$set": module}
		_, err := coll.UpdateOne(ctx, filter, update)
		return err
	}

	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	filter := bson.M{"name": module.Name}
	update := bson.M{"$set": module}
	_, err = coll.UpdateOne(ctx, filter, update)
	return err
}

func (r *ModuleRepositoryImpl) Delete(ctx context.Context, name string, userID string) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"deleted_by": userID,
		},
	}
	filter := bson.M{"name": name}
	_, err = coll.UpdateOne(ctx, filter, update)
	return err
}

func (r *ModuleRepositoryImpl) FindUsingLookup(ctx context.Context, targetModule string) ([]models.Entity, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"deleted_at": bson.M{"$exists": false},
		"fields": bson.M{
			"$elemMatch": bson.M{
				"type":          "lookup",
				"lookup.module": targetModule,
			},
		},
	}

	cursor, err := coll.Find(ctx, filter)
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
	// Note: We need to run this on the Tenant DB collection when initializing.
	// This method might need to be called explicitly for each new tenant DB or we lazily assume?
	// For now, keeping it here.

	// Since we can't get collection without context, this method as "EnsureIndexes(ctx)" assumes ctx has TenantID.
	// Helper will fail if no tenantID.

	coll, err := r.getCollection(ctx)
	if err != nil {
		return err // Or nil if likely no context
	}

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "name", Value: 1},
			},
			Options: options.Index().SetName("idx_name").SetUnique(true),
		},
	}
	_, err = coll.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *ModuleRepositoryImpl) EnsureGlobalIndexes(ctx context.Context) error {
	db := r.DB.GetControlPlaneDB()
	coll := db.Collection("default_modules")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "name", Value: 1},
			},
			Options: options.Index().SetName("idx_template_name").SetUnique(true),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
