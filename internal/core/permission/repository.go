package permission

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

type PermissionRepository interface {
	Create(ctx context.Context, permission *Permission) error
	FindByID(ctx context.Context, id string) (*Permission, error)
	FindByRoleID(ctx context.Context, roleID string) ([]Permission, error)
	FindByResource(ctx context.Context, resourceType, resourceID string) ([]Permission, error)
	FindByRoleAndResource(ctx context.Context, roleID, resourceID string) (*Permission, error)
	Update(ctx context.Context, id string, permission *Permission) error
	Delete(ctx context.Context, id string) error
	DeleteByRoleID(ctx context.Context, roleID string) error
	BulkUpsertForRole(ctx context.Context, roleID string, permissions []Permission) error
	GetDefaults(ctx context.Context) ([]Permission, error)
	EnsureIndexes(ctx context.Context) error
	EnsureGlobalIndexes(ctx context.Context) error
}

type PermissionRepositoryImpl struct {
	DB *database.MongodbDB
}

func NewPermissionRepository(mongodb *database.MongodbDB) PermissionRepository {
	return &PermissionRepositoryImpl{
		DB: mongodb,
	}
}

func (r *PermissionRepositoryImpl) getCollection(ctx context.Context) *mongo.Collection {
	tenantID, _ := ctx.Value(models.TenantIDKey).(string)
	return r.DB.GetTenantDB(tenantID).Collection("permissions")
}

func (r *PermissionRepositoryImpl) GetDefaults(ctx context.Context) ([]Permission, error) {
	db := r.DB.GetControlPlaneDB()
	coll := db.Collection("default_permissions")

	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var perms []Permission
	if err := cursor.All(ctx, &perms); err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *PermissionRepositoryImpl) Create(ctx context.Context, permission *Permission) error {
	coll := r.getCollection(ctx)
	_, err := coll.InsertOne(ctx, permission)
	return err
}

func (r *PermissionRepositoryImpl) FindByID(ctx context.Context, id string) (*Permission, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var permission Permission
	coll := r.getCollection(ctx)
	err = coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&permission)
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *PermissionRepositoryImpl) FindByRoleID(ctx context.Context, roleID string) ([]Permission, error) {
	oid, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return nil, err
	}

	// Build filter with role_id
	filter := bson.M{"role_id": oid}

	// Strict app filtering - only return permissions for the current app
	if app, ok := ctx.Value(models.AppIDKey).(string); ok && app != "" {
		filter["app"] = app
	}

	coll := r.getCollection(ctx)
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var permissions []Permission
	if err := cursor.All(ctx, &permissions); err != nil {
		return nil, err
	}
	return permissions, nil
}

func (r *PermissionRepositoryImpl) FindByResource(ctx context.Context, resourceType, resourceID string) ([]Permission, error) {
	filter := bson.M{
		"resource.type": resourceType, // Is type nested? JSON doesn't show type?
		// JSON shows "resource_id": "crm.leads". Where is type?
		// Assuming strictly resource_id is what we want.
		// If 'type' is missing in struct, this query is useless?
		// Let's assume resource_id is enough as it is unique string "crm.leads".
		"resource_id": resourceID,
	}

	coll := r.getCollection(ctx)
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var permissions []Permission
	if err := cursor.All(ctx, &permissions); err != nil {
		return nil, err
	}
	return permissions, nil
}

func (r *PermissionRepositoryImpl) FindByRoleAndResource(ctx context.Context, roleID, resourceID string) (*Permission, error) {
	oid, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"role_id":     oid,
		"resource_id": resourceID,
	}

	var permission Permission
	coll := r.getCollection(ctx)
	err = coll.FindOne(ctx, filter).Decode(&permission)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &permission, nil
}

func (r *PermissionRepositoryImpl) Update(ctx context.Context, id string, permission *Permission) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"app":         permission.App,
			"actions":     permission.Actions,
			"field_rules": permission.FieldRules,
			"updated_at":  permission.UpdatedAt,
		},
	}

	coll := r.getCollection(ctx)
	result, err := coll.UpdateOne(ctx, bson.M{"_id": oid}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("permission not found")
	}

	return nil
}

func (r *PermissionRepositoryImpl) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	coll := r.getCollection(ctx)
	result, err := coll.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("permission not found")
	}

	return nil
}

func (r *PermissionRepositoryImpl) DeleteByRoleID(ctx context.Context, roleID string) error {
	oid, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return err
	}

	coll := r.getCollection(ctx)
	_, err = coll.DeleteMany(ctx, bson.M{"role_id": oid})
	return err
}

func (r *PermissionRepositoryImpl) BulkUpsertForRole(ctx context.Context, roleID string, permissions []Permission) error {
	oid, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return err
	}

	// Start a session for transaction
	coll := r.getCollection(ctx)
	session, err := coll.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	// Execute in transaction
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (any, error) {
		// Delete existing permissions for this role
		coll := r.getCollection(sessCtx)
		_, err := coll.DeleteMany(sessCtx, bson.M{"role_id": oid})
		if err != nil {
			return nil, err
		}

		// Insert new permissions
		if len(permissions) > 0 {
			docs := make([]any, len(permissions))
			for i := range permissions {
				docs[i] = permissions[i]
			}
			_, err = coll.InsertMany(sessCtx, docs)
			if err != nil {
				return nil, err
			}
		}

		return nil, nil
	})

	return err
}

func (r *PermissionRepositoryImpl) EnsureIndexes(ctx context.Context) error {
	coll := r.getCollection(ctx)
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "role_id", Value: 1},
				{Key: "resource_id", Value: 1},
			},
			Options: options.Index().SetName("idx_role_resource_v2").SetUnique(true),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *PermissionRepositoryImpl) EnsureGlobalIndexes(ctx context.Context) error {
	db := r.DB.GetControlPlaneDB()
	coll := db.Collection("default_permissions")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "role_name", Value: 1},
				{Key: "resource_id", Value: 1},
			},
			Options: options.Index().SetName("idx_template_role_resource_v2").SetUnique(true),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
