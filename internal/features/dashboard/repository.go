package dashboard

import (
	"context"
	"errors"
	"time"

	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DashboardRepository interface {
	Create(ctx context.Context, dashboard *DashboardConfig) error
	Get(ctx context.Context, id string) (*DashboardConfig, error)
	FindByUserID(ctx context.Context, userID string) ([]DashboardConfig, error)
	Update(ctx context.Context, id string, dashboard *DashboardConfig) error
	Delete(ctx context.Context, id string) error
	GetDefaultByUserID(ctx context.Context, userID string) (*DashboardConfig, error)
	SetDefault(ctx context.Context, userID string, dashboardID string) error
}

type DashboardRepositoryImpl struct {
	collection *mongo.Collection
}

func NewDashboardRepository(db *database.MongodbDB) DashboardRepository {
	return &DashboardRepositoryImpl{
		collection: db.DB.Collection("dashboards"),
	}
}

func (r *DashboardRepositoryImpl) Create(ctx context.Context, dashboard *DashboardConfig) error {
	if dashboard.ID.IsZero() {
		dashboard.ID = primitive.NewObjectID()
	}
	dashboard.CreatedAt = time.Now()
	dashboard.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, dashboard)
	return err
}

func (r *DashboardRepositoryImpl) Get(ctx context.Context, id string) (*DashboardConfig, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var dashboard DashboardConfig
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&dashboard)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("dashboard not found")
		}
		return nil, err
	}
	return &dashboard, nil
}

func (r *DashboardRepositoryImpl) FindByUserID(ctx context.Context, userID string) ([]DashboardConfig, error) {
	oid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"$or": []bson.M{
			{"user_id": oid},
			{"is_shared": true},
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var dashboards []DashboardConfig
	if err = cursor.All(ctx, &dashboards); err != nil {
		return nil, err
	}

	return dashboards, nil
}

func (r *DashboardRepositoryImpl) Update(ctx context.Context, id string, dashboard *DashboardConfig) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	dashboard.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"name":        dashboard.Name,
			"description": dashboard.Description,
			"is_default":  dashboard.IsDefault,
			"is_shared":   dashboard.IsShared,
			"widgets":     dashboard.Widgets,
			"layout":      dashboard.Layout,
			"updated_at":  dashboard.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("dashboard not found")
	}

	return nil
}

func (r *DashboardRepositoryImpl) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("dashboard not found")
	}

	return nil
}

func (r *DashboardRepositoryImpl) GetDefaultByUserID(ctx context.Context, userID string) (*DashboardConfig, error) {
	oid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	var dashboard DashboardConfig
	err = r.collection.FindOne(ctx, bson.M{
		"user_id":    oid,
		"is_default": true,
	}).Decode(&dashboard)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &dashboard, nil
}

func (r *DashboardRepositoryImpl) SetDefault(ctx context.Context, userID string, dashboardID string) error {
	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	dashboardOID, err := primitive.ObjectIDFromHex(dashboardID)
	if err != nil {
		return err
	}

	_, err = r.collection.UpdateMany(ctx,
		bson.M{"user_id": userOID},
		bson.M{"$set": bson.M{"is_default": false}},
	)
	if err != nil {
		return err
	}

	result, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": dashboardOID, "user_id": userOID},
		bson.M{"$set": bson.M{"is_default": true}},
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("dashboard not found or does not belong to user")
	}

	return nil
}
