package approval

import (
	"context"
	"go-crm/internal/database"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ApprovalRepository interface {
	Create(ctx context.Context, workflow ApprovalWorkflow) error
	GetByModuleID(ctx context.Context, moduleID string) (*ApprovalWorkflow, error)
	ListActiveByModuleID(ctx context.Context, moduleID string) ([]ApprovalWorkflow, error)
	GetByID(ctx context.Context, id string) (*ApprovalWorkflow, error)
	List(ctx context.Context) ([]ApprovalWorkflow, error)
	Update(ctx context.Context, id string, workflow ApprovalWorkflow) error
	Delete(ctx context.Context, id string) error
}

type ApprovalRepositoryImpl struct {
	Collection *mongo.Collection
}

func NewApprovalRepository(mongodb *database.MongodbDB) ApprovalRepository {
	return &ApprovalRepositoryImpl{
		Collection: mongodb.DB.Collection("approval_workflows"),
	}
}

func (r *ApprovalRepositoryImpl) Create(ctx context.Context, workflow ApprovalWorkflow) error {
	_, err := r.Collection.InsertOne(ctx, workflow)
	return err
}

func (r *ApprovalRepositoryImpl) GetByModuleID(ctx context.Context, moduleID string) (*ApprovalWorkflow, error) {
	var workflow ApprovalWorkflow
	err := r.Collection.FindOne(ctx, bson.M{"module_id": moduleID, "active": true}).Decode(&workflow)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No active workflow found for this module
		}
		return nil, err
	}
	return &workflow, nil
}

func (r *ApprovalRepositoryImpl) ListActiveByModuleID(ctx context.Context, moduleID string) ([]ApprovalWorkflow, error) {
	cursor, err := r.Collection.Find(ctx, bson.M{"module_id": moduleID, "active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var workflows []ApprovalWorkflow
	if err = cursor.All(ctx, &workflows); err != nil {
		return nil, err
	}
	return workflows, nil
}

func (r *ApprovalRepositoryImpl) GetByID(ctx context.Context, id string) (*ApprovalWorkflow, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var workflow ApprovalWorkflow
	err = r.Collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&workflow)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &workflow, nil
}

func (r *ApprovalRepositoryImpl) List(ctx context.Context) ([]ApprovalWorkflow, error) {
	cursor, err := r.Collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var workflows []ApprovalWorkflow
	if err = cursor.All(ctx, &workflows); err != nil {
		return nil, err
	}
	return workflows, nil
}

func (r *ApprovalRepositoryImpl) Update(ctx context.Context, id string, workflow ApprovalWorkflow) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	update := bson.M{
		"$set": bson.M{
			"name":       workflow.Name,
			"active":     workflow.Active,
			"criteria":   workflow.Criteria,
			"steps":      workflow.Steps,
			"updated_at": time.Now(),
		},
	}
	_, err = r.Collection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	return err
}

func (r *ApprovalRepositoryImpl) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.Collection.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
