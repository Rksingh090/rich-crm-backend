package repository

import (
	"context"
	"errors"
	"time"

	"go-crm/internal/database"
	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// SLAPolicyRepository defines the interface for SLA policy operations
type SLAPolicyRepository interface {
	Create(ctx context.Context, policy *models.SLAPolicy) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.SLAPolicy, error)
	FindAll(ctx context.Context) ([]models.SLAPolicy, error)
	FindByPriority(ctx context.Context, priority models.TicketPriority) (*models.SLAPolicy, error)
	Update(ctx context.Context, id primitive.ObjectID, updates bson.M) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

// SLAPolicyRepositoryImpl implements SLAPolicyRepository
type SLAPolicyRepositoryImpl struct {
	collection *mongo.Collection
}

// NewSLAPolicyRepository creates a new SLA policy repository
func NewSLAPolicyRepository(db *database.MongodbDB) SLAPolicyRepository {
	return &SLAPolicyRepositoryImpl{
		collection: db.DB.Collection("sla_policies"),
	}
}

// Create inserts a new SLA policy
func (r *SLAPolicyRepositoryImpl) Create(ctx context.Context, policy *models.SLAPolicy) error {
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, policy)
	if err != nil {
		return err
	}

	policy.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByID retrieves an SLA policy by ID
func (r *SLAPolicyRepositoryImpl) FindByID(ctx context.Context, id primitive.ObjectID) (*models.SLAPolicy, error) {
	var policy models.SLAPolicy
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&policy)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("SLA policy not found")
		}
		return nil, err
	}
	return &policy, nil
}

// FindAll retrieves all SLA policies
func (r *SLAPolicyRepositoryImpl) FindAll(ctx context.Context) ([]models.SLAPolicy, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var policies []models.SLAPolicy
	if err = cursor.All(ctx, &policies); err != nil {
		return nil, err
	}

	return policies, nil
}

// FindByPriority retrieves the active SLA policy for a specific priority
func (r *SLAPolicyRepositoryImpl) FindByPriority(ctx context.Context, priority models.TicketPriority) (*models.SLAPolicy, error) {
	var policy models.SLAPolicy
	err := r.collection.FindOne(ctx, bson.M{
		"priority":  priority,
		"is_active": true,
	}).Decode(&policy)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // No policy found is not an error
		}
		return nil, err
	}
	return &policy, nil
}

// Update updates an SLA policy
func (r *SLAPolicyRepositoryImpl) Update(ctx context.Context, id primitive.ObjectID, updates bson.M) error {
	updates["updated_at"] = time.Now()

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("SLA policy not found")
	}

	return nil
}

// Delete removes an SLA policy
func (r *SLAPolicyRepositoryImpl) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("SLA policy not found")
	}

	return nil
}
