package bulk_operation

import (
	"context"
	"go-crm/internal/database"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BulkOperationRepository interface {
	Create(ctx context.Context, tenantID string, op *BulkOperation) error
	Get(ctx context.Context, tenantID string, id string) (*BulkOperation, error)
	Update(ctx context.Context, tenantID string, op *BulkOperation) error
	FindByUserID(ctx context.Context, tenantID string, userID string, limit int) ([]BulkOperation, error)
	UpdateStatus(ctx context.Context, tenantID string, id string, status BulkOperationStatus) error
}

type BulkOperationRepositoryImpl struct {
	dbManager *database.MongodbDB
}

func NewBulkOperationRepository(dbManager *database.MongodbDB) BulkOperationRepository {
	return &BulkOperationRepositoryImpl{
		dbManager: dbManager,
	}
}

func (r *BulkOperationRepositoryImpl) Create(ctx context.Context, tenantID string, op *BulkOperation) error {
	db := r.dbManager.GetTenantDB(tenantID)
	collection := db.Collection("bulk_operations")

	if op.ID.IsZero() {
		op.ID = primitive.NewObjectID()
	}

	// Set TenantID
	tenantObjID, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}
	op.TenantID = tenantObjID

	op.CreatedAt = time.Now()
	op.UpdatedAt = time.Now()
	op.Status = BulkStatusPending

	_, err = collection.InsertOne(ctx, op)
	return err
}

func (r *BulkOperationRepositoryImpl) Get(ctx context.Context, tenantID string, id string) (*BulkOperation, error) {
	db := r.dbManager.GetTenantDB(tenantID)
	collection := db.Collection("bulk_operations")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	tenantObjID, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	var op BulkOperation
	err = collection.FindOne(ctx, bson.M{
		"_id":       objID,
		"tenant_id": tenantObjID,
	}).Decode(&op)
	if err != nil {
		return nil, err
	}

	return &op, nil
}

func (r *BulkOperationRepositoryImpl) Update(ctx context.Context, tenantID string, op *BulkOperation) error {
	db := r.dbManager.GetTenantDB(tenantID)
	collection := db.Collection("bulk_operations")

	op.UpdatedAt = time.Now()

	tenantObjID, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}

	_, err = collection.ReplaceOne(ctx, bson.M{
		"_id":       op.ID,
		"tenant_id": tenantObjID,
	}, op)
	return err
}

func (r *BulkOperationRepositoryImpl) FindByUserID(ctx context.Context, tenantID string, userID string, limit int) ([]BulkOperation, error) {
	db := r.dbManager.GetTenantDB(tenantID)
	collection := db.Collection("bulk_operations")

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	tenantObjID, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return nil, err
	}

	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := collection.Find(ctx, bson.M{
		"user_id":   objID,
		"tenant_id": tenantObjID,
	}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var ops []BulkOperation
	if err = cursor.All(ctx, &ops); err != nil {
		return nil, err
	}

	return ops, nil
}

func (r *BulkOperationRepositoryImpl) UpdateStatus(ctx context.Context, tenantID string, id string, status BulkOperationStatus) error {
	db := r.dbManager.GetTenantDB(tenantID)
	collection := db.Collection("bulk_operations")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	tenantObjID, err := primitive.ObjectIDFromHex(tenantID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	if status == BulkStatusCompleted || status == BulkStatusFailed {
		now := time.Now()
		update = bson.M{
			"$set": bson.M{
				"status":       status,
				"updated_at":   time.Now(),
				"completed_at": &now,
			},
		}
	}

	_, err = collection.UpdateOne(ctx, bson.M{
		"_id":       objID,
		"tenant_id": tenantObjID,
	}, update)
	return err
}
