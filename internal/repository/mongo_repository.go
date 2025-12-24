package repository

import (
	"context"
	"slices"

	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoUserRepository struct {
	Collection *mongo.Collection
}

func NewMongoUserRepository(db *mongo.Database) *MongoUserRepository {
	return &MongoUserRepository{
		Collection: db.Collection("users"),
	}
}

func (r *MongoUserRepository) Create(ctx context.Context, user *models.User) error {
	_, err := r.Collection.InsertOne(ctx, user)
	return err
}

func (r *MongoUserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.Collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *MongoUserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var user models.User
	err = r.Collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *MongoUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.Collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *MongoUserRepository) List(ctx context.Context, filter map[string]interface{}, limit, offset int64) ([]models.User, int64, error) {
	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(limit)
	}
	if offset > 0 {
		opts.SetSkip(offset)
	}
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.Collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	total, err := r.Collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *MongoUserRepository) Update(ctx context.Context, id string, user *models.User) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"username":   user.Username,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"phone":      user.Phone,
			"status":     user.Status,
			"roles":      user.Roles,
			"updated_at": user.UpdatedAt,
		},
	}

	if user.LastLogin != nil {
		update["$set"].(bson.M)["last_login"] = user.LastLogin
	}

	_, err = r.Collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *MongoUserRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.Collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

type MongoRoleRepository struct {
	Collection *mongo.Collection
}

func NewMongoRoleRepository(db *mongo.Database) *MongoRoleRepository {
	return &MongoRoleRepository{
		Collection: db.Collection("roles"),
	}
}

func (r *MongoRoleRepository) Create(ctx context.Context, role *models.Role) error {
	_, err := r.Collection.InsertOne(ctx, role)
	return err
}

func (r *MongoRoleRepository) FindByName(ctx context.Context, name string) (*models.Role, error) {
	var role models.Role
	err := r.Collection.FindOne(ctx, bson.M{"name": name}).Decode(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *MongoRoleRepository) FindPermissionsByRoleIDs(ctx context.Context, roleIDs []interface{}) ([]string, error) {
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
