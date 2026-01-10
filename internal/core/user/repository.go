package user

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

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmailGlobal(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	List(ctx context.Context, filter map[string]interface{}, limit, offset int64) ([]models.User, int64, error)
	Update(ctx context.Context, id string, user *models.User) error
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) ([]models.User, error)
	EnsureIndexes(ctx context.Context) error
}

type UserRepositoryImpl struct {
	Collection *mongo.Collection
}

func NewUserRepository(mongodb *database.MongodbDB) UserRepository {
	return &UserRepositoryImpl{
		Collection: mongodb.DB.Collection("users"),
	}
}

func (r *UserRepositoryImpl) EnsureIndexes(ctx context.Context) error {
	// Create Global Unique Email Index
	_, err := r.Collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

func (r *UserRepositoryImpl) Create(ctx context.Context, user *models.User) error {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		// Allow creation without tenant context if explicit system flag or strictly require?
		// For global users, we pass "000000000000000000000000" as text.
		return fmt.Errorf("tenant context missing")
	}

	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}
	user.TenantID = oid

	_, err = r.Collection.InsertOne(ctx, user)
	return err
}

func (r *UserRepositoryImpl) FindByEmailGlobal(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	// No org filter, used for login
	err := r.Collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) FindByID(ctx context.Context, id string) (*models.User, error) {
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
	var user models.User
	err = r.Collection.FindOne(ctx, bson.M{"_id": objectID, "tenant_id": oid}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = r.Collection.FindOne(ctx, bson.M{"email": email, "tenant_id": oid}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) List(ctx context.Context, filter map[string]interface{}, limit, offset int64) ([]models.User, int64, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, 0, fmt.Errorf("tenant context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, 0, err
	}
	filter["tenant_id"] = oid

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

	// Decode into maps first to handle invalid data types (e.g. string roles)
	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, 0, err
	}

	var users []models.User
	for _, doc := range results {
		// Fix roles if present and invalid
		if rolesRaw, ok := doc["roles"]; ok {
			var safeRoles []primitive.ObjectID
			if rolesSlice, ok := rolesRaw.(primitive.A); ok {
				for _, r := range rolesSlice {
					if oid, ok := r.(primitive.ObjectID); ok {
						safeRoles = append(safeRoles, oid)
					} else if str, ok := r.(string); ok {
						if oid, err := primitive.ObjectIDFromHex(str); err == nil {
							safeRoles = append(safeRoles, oid)
						}
					}
				}
			}
			doc["roles"] = safeRoles
		}

		// Fix groups if present and invalid
		if groupsRaw, ok := doc["groups"]; ok {
			var safeGroups []primitive.ObjectID
			if groupsSlice, ok := groupsRaw.(primitive.A); ok {
				for _, g := range groupsSlice {
					if oid, ok := g.(primitive.ObjectID); ok {
						safeGroups = append(safeGroups, oid)
					} else if str, ok := g.(string); ok {
						if oid, err := primitive.ObjectIDFromHex(str); err == nil {
							safeGroups = append(safeGroups, oid)
						}
					}
				}
			}
			doc["groups"] = safeGroups
		}

		// Marshal back to bytes then unmarshal to struct
		data, err := bson.Marshal(doc)
		if err != nil {
			continue
		}
		var user models.User
		if err := bson.Unmarshal(data, &user); err == nil {
			users = append(users, user)
		}
	}

	total, err := r.Collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *UserRepositoryImpl) Update(ctx context.Context, id string, user *models.User) error {
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

	_, err = r.Collection.UpdateOne(ctx, bson.M{"_id": objectID, "tenant_id": oid}, update)
	return err
}

func (r *UserRepositoryImpl) Delete(ctx context.Context, id string) error {
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

func (r *UserRepositoryImpl) FindByIDs(ctx context.Context, ids []string) ([]models.User, error) {
	var objectIDs []primitive.ObjectID
	for _, id := range ids {
		if oid, err := primitive.ObjectIDFromHex(id); err == nil {
			objectIDs = append(objectIDs, oid)
		}
	}

	if len(objectIDs) == 0 {
		return []models.User{}, nil
	}

	cursor, err := r.Collection.Find(ctx, bson.M{"_id": bson.M{"$in": objectIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}
