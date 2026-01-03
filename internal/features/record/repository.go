package record

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

type RecordRepository interface {
	Create(ctx context.Context, moduleName string, product models.Product, data map[string]any) (any, error)
	Get(ctx context.Context, moduleName, id string) (map[string]any, error)
	List(ctx context.Context, moduleName string, filter map[string]any, accessFilter map[string]any, limit, offset int64, sortBy string, sortOrder int) ([]map[string]any, error)
	Count(ctx context.Context, moduleName string, filter map[string]any, accessFilter map[string]any) (int64, error)
	Update(ctx context.Context, moduleName, id string, data map[string]any) error
	Delete(ctx context.Context, moduleName, id string, userID primitive.ObjectID) error
	Aggregate(ctx context.Context, moduleName string, pipeline mongo.Pipeline) ([]map[string]any, error)
}

type RecordRepositoryImpl struct {
	Collection *mongo.Collection
}

func NewRecordRepository(mongodb *database.MongodbDB) RecordRepository {
	return &RecordRepositoryImpl{
		Collection: mongodb.DB.Collection("entity_records"),
	}
}

func (r *RecordRepositoryImpl) Create(ctx context.Context, moduleName string, product models.Product, data map[string]interface{}) (interface{}, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	record := models.EntityRecord{
		ID:        primitive.NewObjectID(),
		TenantID:  oid,
		Product:   product,
		Entity:    moduleName,
		Data:      data,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Deleted:   false,
	}

	// Capture CreatedBy from context if available (assuming generic UserID key)
	if userID, ok := ctx.Value("user_id").(string); ok {
		record.CreatedBy = userID
		record.UpdatedBy = userID
	}

	_, err = r.Collection.InsertOne(ctx, record)
	if err != nil {
		return nil, err
	}
	return record.ID, nil
}

func (r *RecordRepositoryImpl) Get(ctx context.Context, moduleName, id string) (map[string]interface{}, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	recordID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var record models.EntityRecord
	err = r.Collection.FindOne(ctx, bson.M{"_id": recordID, "tenant_id": oid, "entity": moduleName, "deleted": bson.M{"$ne": true}}).Decode(&record)
	if err != nil {
		return nil, err
	}

	return r.flattenRecord(&record), nil
}

func (r *RecordRepositoryImpl) List(ctx context.Context, moduleName string, filter map[string]any, accessFilter map[string]any, limit, offset int64, sortBy string, sortOrder int) ([]map[string]any, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	// Base filter
	baseQuery := bson.M{
		"tenant_id": oid,
		"entity":    moduleName,
		"deleted":   bson.M{"$ne": true},
	}

	// User Filters (need to map fields to data.field)
	userQuery := bson.M{}
	for k, v := range filter {
		// If key is system field, use as is, else prepend data.
		if k == "_id" || k == "created_at" || k == "updated_at" || k == "created_by" {
			userQuery[k] = v
		} else {
			userQuery["data."+k] = v
		}
	}

	// Combine: Base AND (UserQuery AND AccessFilter)
	// But UserQuery might be empty, AccessFilter might be empty
	andConditions := []bson.M{baseQuery}

	if len(userQuery) > 0 {
		andConditions = append(andConditions, userQuery)
	}
	if len(accessFilter) > 0 {
		andConditions = append(andConditions, accessFilter)
	}

	finalQuery := bson.M{"$and": andConditions}

	findOptions := options.Find()
	findOptions.SetLimit(limit)
	findOptions.SetSkip(offset)

	// Sort logic
	if sortBy == "" {
		sortBy = "created_at"
	}
	if sortOrder == 0 {
		sortOrder = -1
	}

	sortKey := sortBy
	if sortBy != "_id" && sortBy != "created_at" && sortBy != "updated_at" {
		sortKey = "data." + sortBy
	}

	findOptions.SetSort(bson.D{{Key: sortKey, Value: sortOrder}})

	cursor, err := r.Collection.Find(ctx, finalQuery, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.EntityRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	results := make([]map[string]any, len(records))
	for i, rec := range records {
		results[i] = r.flattenRecord(&rec)
	}
	return results, nil
}

func (r *RecordRepositoryImpl) Update(ctx context.Context, moduleName, id string, data map[string]any) error {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}

	recordID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	// Flatten update: map fields to data.field
	updateSet := bson.M{
		"updated_at": time.Now(),
	}
	for k, v := range data {
		updateSet["data."+k] = v
	}
	// TODO: Handle UpdatedBy

	_, err = r.Collection.UpdateOne(ctx, bson.M{"_id": recordID, "tenant_id": oid, "entity": moduleName}, bson.M{"$set": updateSet})
	return err
}

func (r *RecordRepositoryImpl) Delete(ctx context.Context, moduleName, id string, userID primitive.ObjectID) error {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}

	recordID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"deleted":    true,
			"deleted_at": time.Now(),
			"deleted_by": userID.Hex(),
		},
	}

	_, err = r.Collection.UpdateOne(ctx, bson.M{"_id": recordID, "tenant_id": oid, "entity": moduleName}, update)
	return err
}

func (r *RecordRepositoryImpl) Count(ctx context.Context, moduleName string, filter map[string]any, accessFilter map[string]any) (int64, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return 0, fmt.Errorf("organization context missing")
	}
	oid, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return 0, err
	}

	baseQuery := bson.M{
		"tenant_id": oid,
		"entity":    moduleName,
		"deleted":   bson.M{"$ne": true},
	}

	userQuery := bson.M{}
	for k, v := range filter {
		if k == "_id" || k == "created_at" || k == "updated_at" {
			userQuery[k] = v
		} else {
			userQuery["data."+k] = v
		}
	}

	andConditions := []bson.M{baseQuery}
	if len(userQuery) > 0 {
		andConditions = append(andConditions, userQuery)
	}
	if len(accessFilter) > 0 {
		andConditions = append(andConditions, accessFilter)
	}
	finalQuery := bson.M{"$and": andConditions}

	return r.Collection.CountDocuments(ctx, finalQuery)
}

func (r *RecordRepositoryImpl) Aggregate(ctx context.Context, moduleName string, pipeline mongo.Pipeline) ([]map[string]any, error) {
	// Aggregation is tricky because of data nesting. Caller likely sends pipeline for flat structure.
	// For now, assume pipeline is adjusted or basic support.
	// We should probably inject a $match stage for tenant_id and entity at the start.

	// This is a placeholder as full aggregation support on nested data requires rewriting the pipeline
	// which is complex. For basic use cases, we might encourage List usage.

	return nil, fmt.Errorf("aggregation not yet supported on unified collection")
}

func (r *RecordRepositoryImpl) flattenRecord(rec *models.EntityRecord) map[string]any {
	flat := make(map[string]any)
	for k, v := range rec.Data {
		flat[k] = v
	}
	flat["_id"] = rec.ID
	flat["id"] = rec.ID // convenience
	flat["created_at"] = rec.CreatedAt
	flat["updated_at"] = rec.UpdatedAt
	flat["created_by"] = rec.CreatedBy
	flat["updated_by"] = rec.UpdatedBy
	// flat["entity"] = rec.Entity
	// flat["product"] = rec.Product
	return flat
}
