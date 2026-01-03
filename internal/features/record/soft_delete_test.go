package record

import (
	"context"
	"testing"

	common_models "go-crm/internal/common/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// We use mtest for mongo integration testing logic if possible,
// but since we don't have mtest setup in this enviroment easily, we can write a mock test
// or simpler unit test if we can mock the repo.
// Given the constraints, I'll write a test that would run if DB was connected,
// using the same pattern as existing tests if any.
// Since I haven't seen existing repository tests, I'll create a test that depends on a mock repository logic
// or just verify the Query construction logic if I can extract it.
//
// However, since we modified the Repository directly which uses actual Mongo driver,
// testing it without a running Mongo is hard.
// I will instead trust the implementation for now and maybe write a "dry run" test if I could.
//
// Let's write a test that checks if Service calls Repo correctly.

type MockRecordRepo struct {
	CapturedDeleteID string
	CapturedUserID   primitive.ObjectID
	CapturedFilter   map[string]any
}

func (m *MockRecordRepo) Create(ctx context.Context, moduleName string, product common_models.Product, data map[string]any) (any, error) {
	return primitive.NewObjectID(), nil
}
func (m *MockRecordRepo) Get(ctx context.Context, moduleName, id string) (map[string]any, error) {
	return map[string]any{"_id": id}, nil
}
func (m *MockRecordRepo) List(ctx context.Context, moduleName string, filter map[string]any, accessFilter map[string]any, limit, offset int64, sortBy string, sortOrder int) ([]map[string]any, error) {
	m.CapturedFilter = filter
	return []map[string]any{}, nil
}
func (m *MockRecordRepo) Count(ctx context.Context, moduleName string, filter map[string]any, accessFilter map[string]any) (int64, error) {
	return 0, nil
}
func (m *MockRecordRepo) Update(ctx context.Context, moduleName, id string, data map[string]any) error {
	return nil
}
func (m *MockRecordRepo) Delete(ctx context.Context, moduleName, id string, userID primitive.ObjectID) error {
	m.CapturedDeleteID = id
	m.CapturedUserID = userID
	return nil
}
func (m *MockRecordRepo) Aggregate(ctx context.Context, moduleName string, pipeline mongo.Pipeline) ([]map[string]any, error) {
	return nil, nil
}

type MockAuditService struct {
}

func (m *MockAuditService) LogChange(ctx context.Context, action common_models.AuditAction, module string, recordID string, changes map[string]common_models.Change) error {
	return nil
}

func (m *MockAuditService) ListLogs(ctx context.Context, filters map[string]interface{}, page, limit int64) ([]common_models.AuditLog, error) {
	return []common_models.AuditLog{}, nil
}

func TestServiceSoftDeletePassesUserID(t *testing.T) {
	mockRepo := &MockRecordRepo{}
	mockAudit := &MockAuditService{}
	service := &RecordServiceImpl{
		RecordRepo:   mockRepo,
		AuditService: mockAudit,
	}

	userID := primitive.NewObjectID()
	ctx := context.Background()
	moduleName := "contacts"
	id := primitive.NewObjectID().Hex()

	err := service.DeleteRecord(ctx, moduleName, id, userID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if mockRepo.CapturedDeleteID != id {
		t.Errorf("Expected delete ID %s, got %s", id, mockRepo.CapturedDeleteID)
	}
	if mockRepo.CapturedUserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, mockRepo.CapturedUserID)
	}
}
