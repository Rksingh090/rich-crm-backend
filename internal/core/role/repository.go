package role

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

type RoleRepository interface {
	Create(ctx context.Context, role *Role) error
	FindByID(ctx context.Context, id string) (*Role, error)
	FindByName(ctx context.Context, name string) (*Role, error)
	List(ctx context.Context) ([]Role, error)
	Update(ctx context.Context, id string, role *Role) error
	Delete(ctx context.Context, id string) error
	FindPermissionsByRoleIDs(ctx context.Context, roleIDs []any) ([]string, error)
	GetDefaults(ctx context.Context) ([]Role, error)
	EnsureIndexes(ctx context.Context) error
	EnsureGlobalIndexes(ctx context.Context) error
}

type RoleRepositoryImpl struct {
	DB *database.MongodbDB
}

func NewRoleRepository(mongodb *database.MongodbDB) RoleRepository {
	return &RoleRepositoryImpl{
		DB: mongodb,
	}
}

// helper
func (r *RoleRepositoryImpl) getCollection(ctx context.Context) (*mongo.Collection, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	db := r.DB.GetTenantDB(tenantID)
	return db.Collection("roles"), nil
}

func (r *RoleRepositoryImpl) GetDefaults(ctx context.Context) ([]Role, error) {
	// Read from Control Plane
	db := r.DB.GetControlPlaneDB()
	coll := db.Collection("default_roles")

	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var roles []Role
	if err = cursor.All(ctx, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *RoleRepositoryImpl) Create(ctx context.Context, role *Role) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	oid, _ := primitive.ObjectIDFromHex(ctx.Value(models.TenantIDKey).(string))
	role.TenantID = oid

	_, err = coll.InsertOne(ctx, role)
	return err
}

func (r *RoleRepositoryImpl) FindByID(ctx context.Context, id string) (*Role, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var role Role
	// No tenant_id filter needed as we are in tenant DB
	err = coll.FindOne(ctx, bson.M{"_id": objectID}).Decode(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepositoryImpl) FindByName(ctx context.Context, name string) (*Role, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	var role Role
	err = coll.FindOne(ctx, bson.M{"name": name}).Decode(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepositoryImpl) List(ctx context.Context) ([]Role, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var roles []Role
	if err = cursor.All(ctx, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *RoleRepositoryImpl) Update(ctx context.Context, id string, role *Role) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"name":              role.Name,
			"description":       role.Description,
			"field_permissions": role.FieldPermissions,
			"updated_at":        role.UpdatedAt,
		},
	}

	_, err = coll.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *RoleRepositoryImpl) Delete(ctx context.Context, id string) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = coll.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *RoleRepositoryImpl) FindPermissionsByRoleIDs(ctx context.Context, roleIDs []any) ([]string, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": bson.M{"$in": roleIDs}}
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var roles []Role
	if err = cursor.All(ctx, &roles); err != nil {
		return nil, err
	}

	// Note: Permissions are now managed via the Permission collection, not embedded in roles
	// This method returns an empty list as roles no longer have embedded permissions
	return []string{}, nil
}

func (r *RoleRepositoryImpl) EnsureIndexes(ctx context.Context) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "name", Value: 1},
			},
			Options: options.Index().SetName("idx_role_name").SetUnique(true),
		},
	}
	_, err = coll.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *RoleRepositoryImpl) EnsureGlobalIndexes(ctx context.Context) error {
	db := r.DB.GetControlPlaneDB()
	coll := db.Collection("default_roles")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "name", Value: 1},
			},
			Options: options.Index().SetName("idx_template_role_name").SetUnique(true),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}
