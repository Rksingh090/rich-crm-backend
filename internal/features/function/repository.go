package function

import (
	"context"
	"fmt"
	"time"

	"go-crm/internal/common/models"
	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type FunctionRepository interface {
	Create(ctx context.Context, function *Function) error
	GetByID(ctx context.Context, id string) (*Function, error)
	List(ctx context.Context, moduleName string) ([]Function, error)
	Update(ctx context.Context, function *Function) error
	Delete(ctx context.Context, id string) error
}

type FunctionRepositoryImpl struct {
	db *database.MongodbDB
}

func NewFunctionRepository(db *database.MongodbDB) FunctionRepository {
	return &FunctionRepositoryImpl{
		db: db,
	}
}

func (r *FunctionRepositoryImpl) getCollection(ctx context.Context) (*mongo.Collection, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		// Try primitive.ObjectID
		if oid, ok := ctx.Value(models.TenantIDKey).(primitive.ObjectID); ok {
			tenantID = oid.Hex()
		}
	}

	if tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}

	return r.db.GetTenantDB(tenantID).Collection("functions"), nil
}

func (r *FunctionRepositoryImpl) Create(ctx context.Context, function *Function) error {
	function.CreatedAt = time.Now()
	function.UpdatedAt = time.Now()

	if function.ID.IsZero() {
		function.ID = primitive.NewObjectID()
	}

	col, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	_, err = col.InsertOne(ctx, function)
	return err
}

func (r *FunctionRepositoryImpl) GetByID(ctx context.Context, id string) (*Function, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	col, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": oid}

	// Add app check if present in context, but allow functions with no app field
	if appID, ok := ctx.Value(models.AppIDKey).(string); ok && appID != "" {
		filter["$or"] = []bson.M{
			{"app": appID},
			{"app": ""},
			{"app": bson.M{"$exists": false}},
		}
	}

	var function Function
	err = col.FindOne(ctx, filter).Decode(&function)
	if err != nil {
		return nil, err
	}

	return &function, nil
}

func (r *FunctionRepositoryImpl) List(ctx context.Context, moduleName string) ([]Function, error) {
	col, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	filter := bson.M{}

	if moduleName != "" {
		filter["$or"] = []bson.M{
			{"module_name": moduleName},
			{"module_name": ""},
			{"module_name": bson.M{"$exists": false}},
		}
	}

	// Add app check
	if appID, ok := ctx.Value(models.AppIDKey).(string); ok && appID != "" {
		appFilter := []bson.M{
			{"app": appID},
			{"app": ""},
			{"app": bson.M{"$exists": false}},
		}
		if existingOr, ok := filter["$or"]; ok {
			// If we already have an OR for module_name, we need to AND it with the app OR
			filter = bson.M{
				"$and": []bson.M{
					{"$or": existingOr},
					{"$or": appFilter},
				},
			}
		} else {
			filter["$or"] = appFilter
		}
	}

	cursor, err := col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var functions []Function
	if err = cursor.All(ctx, &functions); err != nil {
		return nil, err
	}

	return functions, nil
}

func (r *FunctionRepositoryImpl) Update(ctx context.Context, function *Function) error {
	function.UpdatedAt = time.Now()

	col, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": function.ID}
	update := bson.M{"$set": function}

	_, err = col.UpdateOne(ctx, filter, update)
	return err
}

func (r *FunctionRepositoryImpl) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	col, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	_, err = col.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
