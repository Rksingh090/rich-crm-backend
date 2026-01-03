package bulk_operation

import (
	"context"
	"go-crm/internal/database"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BulkOperationRepository interface {
	Create(ctx context.Context, op *BulkOperation) error
	Get(ctx context.Context, id string) (*BulkOperation, error)
	Update(ctx context.Context, op *BulkOperation) error
	FindByUserID(ctx context.Context, userID string, limit int) ([]BulkOperation, error)
	UpdateStatus(ctx context.Context, id string, status BulkOperationStatus) error
}

type BulkOperationRepositoryImpl struct {
	collection *mongo.Collection
}

func NewBulkOperationRepository(db *database.MongodbDB) BulkOperationRepository {
	return &BulkOperationRepositoryImpl{
		collection: db.DB.Collection("bulk_operations"),
	}
}

func (r *BulkOperationRepositoryImpl) Create(ctx context.Context, op *BulkOperation) error {
	if op.ID.IsZero() {
		op.ID = primitive.NewObjectID()
	}
	op.CreatedAt = time.Now()
	op.UpdatedAt = time.Now()
	op.Status = BulkStatusPending

	_, err := r.collection.InsertOne(ctx, op)
	return err
}

func (r *BulkOperationRepositoryImpl) Get(ctx context.Context, id string) (*BulkOperation, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var op BulkOperation
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&op)
	if err != nil {
		return nil, err
	}

	return &op, nil
}

func (r *BulkOperationRepositoryImpl) Update(ctx context.Context, op *BulkOperation) error {
	op.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": op.ID}, op)
	return err
}

func (r *BulkOperationRepositoryImpl) FindByUserID(ctx context.Context, userID string, limit int) ([]BulkOperation, error) {
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": objID}, opts)
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

func (r *BulkOperationRepositoryImpl) UpdateStatus(ctx context.Context, id string, status BulkOperationStatus) error {
	objID, err := primitive.ObjectIDFromHex(id)
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

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}
