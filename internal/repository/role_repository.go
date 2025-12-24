package repository

import (
	"context"
	"slices"

	"go-crm/internal/database"
	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type RoleRepository interface {
	Create(ctx context.Context, role *models.Role) error
	FindByName(ctx context.Context, name string) (*models.Role, error)
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

func (r *RoleRepositoryImpl) FindByName(ctx context.Context, name string) (*models.Role, error) {
	var role models.Role
	err := r.Collection.FindOne(ctx, bson.M{"name": name}).Decode(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepositoryImpl) FindPermissionsByRoleIDs(ctx context.Context, roleIDs []interface{}) ([]string, error) {
	// roleIDs should be []primitive.ObjectID

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

	var permissions []string
	for _, role := range roles {
		for _, perm := range role.Permissions {
			if !slices.Contains(permissions, perm) {
				permissions = append(permissions, perm)
			}
		}
	}
	return permissions, nil
}
