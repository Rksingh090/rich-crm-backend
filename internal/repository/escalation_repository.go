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

// EscalationRuleRepository defines the interface for escalation rule operations
type EscalationRuleRepository interface {
	Create(ctx context.Context, rule *models.EscalationRule) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.EscalationRule, error)
	FindAll(ctx context.Context) ([]models.EscalationRule, error)
	FindActive(ctx context.Context) ([]models.EscalationRule, error)
	Update(ctx context.Context, id primitive.ObjectID, updates bson.M) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

// EscalationRuleRepositoryImpl implements EscalationRuleRepository
type EscalationRuleRepositoryImpl struct {
	collection *mongo.Collection
}

// NewEscalationRuleRepository creates a new escalation rule repository
func NewEscalationRuleRepository(db *database.MongodbDB) EscalationRuleRepository {
	return &EscalationRuleRepositoryImpl{
		collection: db.DB.Collection("escalation_rules"),
	}
}

// Create inserts a new escalation rule
func (r *EscalationRuleRepositoryImpl) Create(ctx context.Context, rule *models.EscalationRule) error {
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, rule)
	if err != nil {
		return err
	}

	rule.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByID retrieves an escalation rule by ID
func (r *EscalationRuleRepositoryImpl) FindByID(ctx context.Context, id primitive.ObjectID) (*models.EscalationRule, error) {
	var rule models.EscalationRule
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&rule)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("escalation rule not found")
		}
		return nil, err
	}
	return &rule, nil
}

// FindAll retrieves all escalation rules
func (r *EscalationRuleRepositoryImpl) FindAll(ctx context.Context) ([]models.EscalationRule, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rules []models.EscalationRule
	if err = cursor.All(ctx, &rules); err != nil {
		return nil, err
	}

	return rules, nil
}

// FindActive retrieves all active escalation rules
func (r *EscalationRuleRepositoryImpl) FindActive(ctx context.Context) ([]models.EscalationRule, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"is_active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rules []models.EscalationRule
	if err = cursor.All(ctx, &rules); err != nil {
		return nil, err
	}

	return rules, nil
}

// Update updates an escalation rule
func (r *EscalationRuleRepositoryImpl) Update(ctx context.Context, id primitive.ObjectID, updates bson.M) error {
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
		return errors.New("escalation rule not found")
	}

	return nil
}

// Delete removes an escalation rule
func (r *EscalationRuleRepositoryImpl) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("escalation rule not found")
	}

	return nil
}
