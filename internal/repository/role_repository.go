package repository

import (
	"context"
	"slices"

	"go-crm/internal/database"
	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RoleRepository interface {
	Create(ctx context.Context, role *models.Role) error
	FindByID(ctx context.Context, id string) (*models.Role, error)
	FindByName(ctx context.Context, name string) (*models.Role, error)
	List(ctx context.Context) ([]models.Role, error)
	Update(ctx context.Context, id string, role *models.Role) error
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

func (r *RoleRepositoryImpl) Create(ctx context.Context, role *models.Role) error {
	_, err := r.Collection.InsertOne(ctx, role)
	return err
}

func (r *RoleRepositoryImpl) FindByID(ctx context.Context, id string) (*models.Role, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var role models.Role
	err = r.Collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepositoryImpl) FindByName(ctx context.Context, name string) (*models.Role, error) {
	var role models.Role
	err := r.Collection.FindOne(ctx, bson.M{"name": name}).Decode(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepositoryImpl) List(ctx context.Context) ([]models.Role, error) {
	cursor, err := r.Collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var roles []models.Role
	if err = cursor.All(ctx, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *RoleRepositoryImpl) Update(ctx context.Context, id string, role *models.Role) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"name":               role.Name,
			"description":        role.Description,
			"module_permissions": role.ModulePermissions,
			"field_permissions":  role.FieldPermissions,
			"updated_at":         role.UpdatedAt,
		},
	}

	_, err = r.Collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *RoleRepositoryImpl) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.Collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *RoleRepositoryImpl) FindPermissionsByRoleIDs(ctx context.Context, roleIDs []interface{}) ([]string, error) {
	filter := bson.M{"_id": bson.M{"$in": roleIDs}}
	cursor, err := r.Collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var roles []models.Role
	if err = cursor.All(ctx, &roles); err != nil {
		return nil, err
	}

	// Collect all unique permissions from all modules
	var permissions []string
	for _, role := range roles {
		for moduleName, modulePerm := range role.ModulePermissions {
			if modulePerm.Create && !slices.Contains(permissions, moduleName+":create") {
				permissions = append(permissions, moduleName+":create")
			}
			if modulePerm.Read && !slices.Contains(permissions, moduleName+":read") {
				permissions = append(permissions, moduleName+":read")
			}
			if modulePerm.Update && !slices.Contains(permissions, moduleName+":update") {
				permissions = append(permissions, moduleName+":update")
			}
			if modulePerm.Delete && !slices.Contains(permissions, moduleName+":delete") {
				permissions = append(permissions, moduleName+":delete")
			}
		}
	}
	return permissions, nil
}
