package approval

import (
	"context"
	"fmt"
	"go-crm/internal/common/models"
	"go-crm/internal/database"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ApprovalRepository interface {
	Create(ctx context.Context, process *ApprovalProcess) error
	GetByModuleID(ctx context.Context, moduleID string) (*ApprovalProcess, error)
	ListActiveByModuleID(ctx context.Context, moduleID string) ([]ApprovalProcess, error)
	GetByID(ctx context.Context, id string) (*ApprovalProcess, error)
	List(ctx context.Context) ([]ApprovalProcess, error)
	Update(ctx context.Context, id string, process ApprovalProcess) error
	Delete(ctx context.Context, id string) error
}

type ApprovalRepositoryImpl struct {
	DB *database.MongodbDB
}

func NewApprovalRepository(mongodb *database.MongodbDB) ApprovalRepository {
	return &ApprovalRepositoryImpl{
		DB: mongodb,
	}
}

func (r *ApprovalRepositoryImpl) getCollection(ctx context.Context) (*mongo.Collection, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	return r.DB.GetTenantDB(tenantID).Collection("approval_processes"), nil
}

func (r *ApprovalRepositoryImpl) Create(ctx context.Context, process *ApprovalProcess) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}
	result, err := coll.InsertOne(ctx, process)
	if err != nil {
		return err
	}
	process.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ApprovalRepositoryImpl) GetByModuleID(ctx context.Context, moduleID string) (*ApprovalProcess, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}
	var process ApprovalProcess
	err = coll.FindOne(ctx, bson.M{"module_id": moduleID, "active": true}).Decode(&process)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No active process found for this module
		}
		return nil, err
	}
	return &process, nil
}

func (r *ApprovalRepositoryImpl) ListActiveByModuleID(ctx context.Context, moduleID string) ([]ApprovalProcess, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}
	cursor, err := coll.Find(ctx, bson.M{"module_id": moduleID, "active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var processes []ApprovalProcess
	if err = cursor.All(ctx, &processes); err != nil {
		return nil, err
	}
	return processes, nil
}

func (r *ApprovalRepositoryImpl) GetByID(ctx context.Context, id string) (*ApprovalProcess, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var process ApprovalProcess
	err = coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&process)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &process, nil
}

func (r *ApprovalRepositoryImpl) List(ctx context.Context) ([]ApprovalProcess, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}
	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var processes []ApprovalProcess
	if err = cursor.All(ctx, &processes); err != nil {
		return nil, err
	}
	return processes, nil
}

func (r *ApprovalRepositoryImpl) Update(ctx context.Context, id string, process ApprovalProcess) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	update := bson.M{
		"$set": bson.M{
			"name":       process.Name,
			"module_id":  process.ModuleID,
			"active":     process.Active,
			"priority":   process.Priority,
			"criteria":   process.Criteria,
			"steps":      process.Steps,
			"layout":     process.Layout,
			"edges":      process.Edges,
			"updated_at": time.Now(),
		},
	}
	_, err = coll.UpdateOne(ctx, bson.M{"_id": oid}, update)
	return err
}

func (r *ApprovalRepositoryImpl) Delete(ctx context.Context, id string) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = coll.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
