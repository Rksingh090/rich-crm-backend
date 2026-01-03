package saved_filter

import (
	"context"
	"fmt"
	"go-crm/internal/common/models"
	"go-crm/internal/database"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SavedFilterRepository interface {
	Create(ctx context.Context, filter *SavedFilter) error
	Get(ctx context.Context, id string) (*SavedFilter, error)
	Update(ctx context.Context, filter *SavedFilter) error
	Delete(ctx context.Context, id string) error
	FindByUser(ctx context.Context, userID string, moduleName string) ([]SavedFilter, error)
	FindPublic(ctx context.Context, moduleName string) ([]SavedFilter, error)
}

type SavedFilterRepositoryImpl struct {
	collection *mongo.Collection
}

func NewSavedFilterRepository(db *database.MongodbDB) SavedFilterRepository {
	return &SavedFilterRepositoryImpl{
		collection: db.DB.Collection("saved_filters"),
	}
}

func (r *SavedFilterRepositoryImpl) Create(ctx context.Context, filter *SavedFilter) error {
	if filter.ID.IsZero() {
		filter.ID = primitive.NewObjectID()
	}
	tenantIDStr, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantIDStr == "" {
		return fmt.Errorf("tenant ID not found in context")
	}
	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		return fmt.Errorf("invalid tenant ID: %v", err)
	}
	filter.TenantID = tenantID
	filter.CreatedAt = time.Now()
	filter.UpdatedAt = time.Now()

	_, err = r.collection.InsertOne(ctx, filter)
	return err
}

func (r *SavedFilterRepositoryImpl) Get(ctx context.Context, id string) (*SavedFilter, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var filter SavedFilter
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&filter)
	if err != nil {
		return nil, err
	}

	return &filter, nil
}

func (r *SavedFilterRepositoryImpl) Update(ctx context.Context, filter *SavedFilter) error {
	existing, err := r.Get(ctx, filter.ID.Hex())
	if err != nil {
		return err
	}

	// Preserve immutable fields
	filter.TenantID = existing.TenantID
	filter.CreatedAt = existing.CreatedAt

	// Preserve UserID if not provided (though usually it shouldn't change)
	if filter.UserID.IsZero() {
		filter.UserID = existing.UserID
	}

	filter.UpdatedAt = time.Now()
	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": filter.ID}, filter)
	return err
}

func (r *SavedFilterRepositoryImpl) Delete(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}

func (r *SavedFilterRepositoryImpl) FindByUser(ctx context.Context, userID string, moduleName string) ([]SavedFilter, error) {
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	query := bson.M{
		"user_id":     objID,
		"module_name": moduleName,
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var filters []SavedFilter
	if err = cursor.All(ctx, &filters); err != nil {
		return nil, err
	}

	return filters, nil
}

func (r *SavedFilterRepositoryImpl) FindPublic(ctx context.Context, moduleName string) ([]SavedFilter, error) {
	query := bson.M{
		"is_public":   true,
		"module_name": moduleName,
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var filters []SavedFilter
	if err = cursor.All(ctx, &filters); err != nil {
		return nil, err
	}

	return filters, nil
}
