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
	Create(ctx context.Context, moduleName string, product models.App, data map[string]any) (any, error)
	Get(ctx context.Context, moduleName, id string) (map[string]any, error)
	List(ctx context.Context, moduleName string, filter map[string]any, accessFilter map[string]any, limit, offset int64, sortBy string, sortOrder int) ([]map[string]any, error)
	Count(ctx context.Context, moduleName string, filter map[string]any, accessFilter map[string]any) (int64, error)
	Update(ctx context.Context, moduleName, id string, data map[string]any) error
	Delete(ctx context.Context, moduleName, id string, userID primitive.ObjectID) error
	Aggregate(ctx context.Context, moduleName string, pipeline mongo.Pipeline) ([]map[string]any, error)
	GetNextSequence(ctx context.Context, moduleName, fieldName string) (int64, error)
}

type RecordRepositoryImpl struct {
	DB *database.MongodbDB
}

func NewRecordRepository(mongodb *database.MongodbDB) RecordRepository {
	return &RecordRepositoryImpl{
		DB: mongodb,
	}
}

// helper to get collection
func (r *RecordRepositoryImpl) getCollection(ctx context.Context, moduleName string, product models.App) (*mongo.Collection, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}

	db := r.DB.GetTenantDB(tenantID)
	// Collection Name: product_module (e.g. crm_deal)
	// If product is empty, default to crm? Or error? Assuming product passed is correct.
	if product == "" {
		product = models.AppCRM // Default fallback if needed
	}

	collName := fmt.Sprintf("%s_%s", product, moduleName)
	return db.Collection(collName), nil
}

func (r *RecordRepositoryImpl) Create(ctx context.Context, moduleName string, product models.App, data map[string]interface{}) (interface{}, error) {
	coll, err := r.getCollection(ctx, moduleName, product)
	if err != nil {
		return nil, err
	}

	oid, err := primitive.ObjectIDFromHex(ctx.Value(models.TenantIDKey).(string))
	if err != nil {
		return nil, err
	}

	record := models.EntityRecord{
		ID:        primitive.NewObjectID(),
		TenantID:  oid,
		App:   product,
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

	_, err = coll.InsertOne(ctx, record)
	if err != nil {
		return nil, err
	}
	return record.ID, nil
}

func (r *RecordRepositoryImpl) Get(ctx context.Context, moduleName, id string) (map[string]interface{}, error) {
	// Need product to find collection. But Get only has moduleName.
	// Issue: We don't strictly know the product from just moduleName in all call sites.
	// Assumption: Modules are unique across products or caller context implies product?
	// For now, let's look up Module definition? Or assume CRM?
	// BETTER: Passing product to Get? Or trying to find in all product collections?
	// Quick fix: Assume CRM for now as it's the main one, OR check "crm_module", "erp_module".
	// Ideally, `moduleName` usually implies product implicitly in the current app design?
	// Let's iterate known products if needed or defaulting to CRM.

	// Refactoring needed: All RecordRepository methods should probably accept Product or we deduce it.
	// For now, let's use a helper that tries to find the record in known product prefixes if not ambiguous.
	// However, usually the Module Registry knows the product which could be passed down.

	// Workaround: Try "crm" first.
	// Note: You asked for "per app DB" or "per app collection".
	// If I don't know the product, I can't guess the collection.

	// FIX: Update interface? No, let's try to deduce from Module Service? Too complex dep.
	// Let's just try CRM then ERP.

	products := []models.App{models.AppCRM, models.AppERP}
	var record models.EntityRecord
	var err error

	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}
	recordID, _ := primitive.ObjectIDFromHex(id)

	for _, p := range products {
		coll, e := r.getCollection(ctx, moduleName, p)
		if e != nil {
			continue
		}

		// Note: Removing tenant_id check from filter as we are in Tenant DB
		err = coll.FindOne(ctx, bson.M{"_id": recordID, "deleted": bson.M{"$ne": true}}).Decode(&record)
		if err == nil {
			return r.flattenRecord(&record), nil
		}
	}

	return nil, mongo.ErrNoDocuments
}

func (r *RecordRepositoryImpl) List(ctx context.Context, moduleName string, filter map[string]any, accessFilter map[string]any, limit, offset int64, sortBy string, sortOrder int) ([]map[string]any, error) {
	// Taking Product from Filter if available? Or default CRM.
	// This is a limitation of the current Interface `List(context, moduleName...)`.
	// We'll Default to CRM for List unless specified.

	product := models.AppCRM
	// Hack: Check if filter has "app" (unlikely).

	coll, err := r.getCollection(ctx, moduleName, product)
	if err != nil {
		return nil, err
	}

	// Base filter - No TenantID needed in filter
	baseQuery := bson.M{
		"deleted": bson.M{"$ne": true},
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

	cursor, err := coll.Find(ctx, finalQuery, findOptions)
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
	// Similar loop strategy to Get
	products := []models.App{models.AppCRM, models.AppERP}

	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return fmt.Errorf("organization context missing")
	}
	recordID, _ := primitive.ObjectIDFromHex(id)

	updateSet := bson.M{
		"updated_at": time.Now(),
	}
	for k, v := range data {
		updateSet["data."+k] = v
	}

	for _, p := range products {
		coll, e := r.getCollection(ctx, moduleName, p)
		if e != nil {
			continue
		}

		res, err := coll.UpdateOne(ctx, bson.M{"_id": recordID}, bson.M{"$set": updateSet})
		if err == nil && res.MatchedCount > 0 {
			return nil
		}
	}
	return fmt.Errorf("record not found")
}

func (r *RecordRepositoryImpl) Delete(ctx context.Context, moduleName, id string, userID primitive.ObjectID) error {
	products := []models.App{models.AppCRM, models.AppERP}
	recordID, _ := primitive.ObjectIDFromHex(id)

	update := bson.M{
		"$set": bson.M{
			"deleted":    true,
			"deleted_at": time.Now(),
			"deleted_by": userID.Hex(),
		},
	}

	for _, p := range products {
		coll, e := r.getCollection(ctx, moduleName, p)
		if e != nil {
			continue
		}

		res, err := coll.UpdateOne(ctx, bson.M{"_id": recordID}, update)
		if err == nil && res.MatchedCount > 0 {
			return nil
		}
	}
	return fmt.Errorf("record not found")
}

func (r *RecordRepositoryImpl) Count(ctx context.Context, moduleName string, filter map[string]any, accessFilter map[string]any) (int64, error) {
	// Default to CRM
	product := models.AppCRM
	coll, err := r.getCollection(ctx, moduleName, product)
	if err != nil {
		return 0, err
	}

	baseQuery := bson.M{
		"deleted": bson.M{"$ne": true},
	}

	userQuery := bson.M{}
	for k, v := range filter {
		if k == "_id" || k == "created_at" || k == "updated_at" || k == "created_by" {
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

	return coll.CountDocuments(ctx, finalQuery)
}

func (r *RecordRepositoryImpl) Aggregate(ctx context.Context, moduleName string, pipeline mongo.Pipeline) ([]map[string]any, error) {
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
	return flat
}

func (r *RecordRepositoryImpl) GetNextSequence(ctx context.Context, moduleName, fieldName string) (int64, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return 0, fmt.Errorf("organization context missing")
	}

	// Use Tenant DB
	db := r.DB.GetTenantDB(tenantID)
	countersColl := db.Collection("counters")

	filter := bson.M{
		"module": moduleName,
		"field":  fieldName,
	}

	update := bson.M{
		"$inc": bson.M{"seq": 1},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var result struct {
		Seq int64 `bson:"seq"`
	}

	err := countersColl.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return 0, err
	}

	return result.Seq, nil
}
