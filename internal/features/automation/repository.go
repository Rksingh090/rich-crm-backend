package automation

import (
	"context"
	"go-crm/internal/database"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AutomationRepository interface {
	Create(ctx context.Context, rule *AutomationRule) error
	GetByID(ctx context.Context, id string) (*AutomationRule, error)
	GetByModule(ctx context.Context, moduleID string) ([]AutomationRule, error)
	List(ctx context.Context) ([]AutomationRule, error)
	Update(ctx context.Context, rule *AutomationRule) error
	Delete(ctx context.Context, id string) error
	Enable(ctx context.Context, id string, active bool) error
}

type AutomationRepositoryImpl struct {
	Collection *mongo.Collection
}

func NewAutomationRepository(mongodb *database.MongodbDB) AutomationRepository {
	return &AutomationRepositoryImpl{
		Collection: mongodb.DB.Collection("automation_rules"),
	}
}

func (r *AutomationRepositoryImpl) Create(ctx context.Context, rule *AutomationRule) error {
	rule.ID = primitive.NewObjectID()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	_, err := r.Collection.InsertOne(ctx, rule)
	return err
}

func (r *AutomationRepositoryImpl) GetByID(ctx context.Context, id string) (*AutomationRule, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var rule AutomationRule
	err = r.Collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&rule)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

func (r *AutomationRepositoryImpl) GetByModule(ctx context.Context, moduleID string) ([]AutomationRule, error) {
	cursor, err := r.Collection.Find(ctx, bson.M{"module_id": moduleID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var rules []AutomationRule
	if err = cursor.All(ctx, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *AutomationRepositoryImpl) List(ctx context.Context) ([]AutomationRule, error) {
	cursor, err := r.Collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var rules []AutomationRule
	if err = cursor.All(ctx, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *AutomationRepositoryImpl) Update(ctx context.Context, rule *AutomationRule) error {
	rule.UpdatedAt = time.Now()
	_, err := r.Collection.UpdateOne(ctx, bson.M{"_id": rule.ID}, bson.M{"$set": rule})
	return err
}

func (r *AutomationRepositoryImpl) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.Collection.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}

func (r *AutomationRepositoryImpl) Enable(ctx context.Context, id string, active bool) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.Collection.UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": bson.M{"active": active, "updated_at": time.Now()}})
	return err
}
