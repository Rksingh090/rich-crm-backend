package role

import (
	"context"
	"fmt"
	"slices"

	"go-crm/internal/common/models"
	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RoleRepository interface {
	Create(ctx context.Context, role *Role) error
	FindByID(ctx context.Context, id string) (*Role, error)
	FindByName(ctx context.Context, name string) (*Role, error)
	List(ctx context.Context) ([]Role, error)
	Update(ctx context.Context, id string, role *Role) error
	Delete(ctx context.Context, id string) error
	FindPermissionsByRoleIDs(ctx context.Context, roleIDs []interface{}) ([]string, error)
}

type RoleRepositoryImpl struct {
	Collection *mongo.Collection
}

func NewRoleRepository(mongodb *database.MongodbDB) RoleRepository {
	return &RoleRepositoryImpl{
		Collection: mongodb.DB.Collection("roles"),
	}
}

func (r *RoleRepositoryImpl) Create(ctx context.Context, role *Role) error {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}
	role.TenantID = oid

	_, err = r.Collection.InsertOne(ctx, role)
	return err
}

func (r *RoleRepositoryImpl) FindByID(ctx context.Context, id string) (*Role, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var role Role
	err = r.Collection.FindOne(ctx, bson.M{"_id": objectID, "tenant_id": oid}).Decode(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepositoryImpl) FindByName(ctx context.Context, name string) (*Role, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	var role Role
	err = r.Collection.FindOne(ctx, bson.M{"name": name, "tenant_id": oid}).Decode(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepositoryImpl) List(ctx context.Context) ([]Role, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	cursor, err := r.Collection.Find(ctx, bson.M{"tenant_id": oid})
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
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
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
			"permissions":       role.Permissions,
			"field_permissions": role.FieldPermissions,
			"updated_at":        role.UpdatedAt,
		},
	}

	_, err = r.Collection.UpdateOne(ctx, bson.M{"_id": objectID, "tenant_id": oid}, update)
	return err
}

func (r *RoleRepositoryImpl) Delete(ctx context.Context, id string) error {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.Collection.DeleteOne(ctx, bson.M{"_id": objectID, "tenant_id": oid})
	return err
}

func (r *RoleRepositoryImpl) FindPermissionsByRoleIDs(ctx context.Context, roleIDs []interface{}) ([]string, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": bson.M{"$in": roleIDs}, "tenant_id": oid}
	cursor, err := r.Collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var roles []Role
	if err = cursor.All(ctx, &roles); err != nil {
		return nil, err
	}

	// Collect all unique permissions from all resources
	var permissions []string
	for _, role := range roles {
		for resourceID, actions := range role.Permissions {
			for action, perm := range actions {
				if perm.Allowed {
					p := resourceID + ":" + action
					if !slices.Contains(permissions, p) {
						permissions = append(permissions, p)
					}
				}
			}
		}
	}
	return permissions, nil
}
