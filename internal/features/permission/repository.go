package permission

import (
	"context"
	"fmt"

	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
}

type PermissionRepositoryImpl struct {
	collection *mongo.Collection
}

func NewPermissionRepository(mongodb *database.MongodbDB) PermissionRepository {
	return &PermissionRepositoryImpl{
		collection: mongodb.DB.Collection("permissions"),
	}
}

func (r *PermissionRepositoryImpl) Create(ctx context.Context, permission *Permission) error {
	_, err := r.collection.InsertOne(ctx, permission)
	return err
}

func (r *PermissionRepositoryImpl) FindByID(ctx context.Context, id string) (*Permission, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var permission Permission
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&permission)
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

	cursor, err := r.collection.Find(ctx, bson.M{"role_id": oid})
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
		"resource.type": resourceType,
		"resource.id":   resourceID,
	}

	cursor, err := r.collection.Find(ctx, filter)
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
		"resource.id": resourceID,
	}

	var permission Permission
	err = r.collection.FindOne(ctx, filter).Decode(&permission)
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
			"actions":     permission.Actions,
			"field_rules": permission.FieldRules,
			"updated_at":  permission.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": oid}, update)
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

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": oid})
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

	_, err = r.collection.DeleteMany(ctx, bson.M{"role_id": oid})
	return err
}

func (r *PermissionRepositoryImpl) BulkUpsertForRole(ctx context.Context, roleID string, permissions []Permission) error {
	oid, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return err
	}

	// Start a session for transaction
	session, err := r.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	// Execute in transaction
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Delete existing permissions for this role
		_, err := r.collection.DeleteMany(sessCtx, bson.M{"role_id": oid})
		if err != nil {
			return nil, err
		}

		// Insert new permissions
		if len(permissions) > 0 {
			docs := make([]interface{}, len(permissions))
			for i := range permissions {
				docs[i] = permissions[i]
			}
			_, err = r.collection.InsertMany(sessCtx, docs)
			if err != nil {
				return nil, err
			}
		}

		return nil, nil
	})

	return err
}
