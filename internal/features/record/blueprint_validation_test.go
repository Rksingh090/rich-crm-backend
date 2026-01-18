package record

import (
	"context"
	"fmt"
	"testing"

	"go-crm/internal/common/models"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/core/role"
	"go-crm/internal/features/webhook"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Mocks

type MockBlueprintValidator struct {
	TargetField string
	Err         error
}

func (m *MockBlueprintValidator) GetActiveBlueprintTargetField(ctx context.Context, module string) (string, error) {
	return m.TargetField, m.Err
}

type MockModuleRepo struct {
	Entity *models.Entity
}

func (m *MockModuleRepo) FindByName(ctx context.Context, name string) (*models.Entity, error) {
	if m.Entity != nil {
		return m.Entity, nil
	}
	return nil, fmt.Errorf("module not found")
}
func (m *MockModuleRepo) Create(ctx context.Context, module *models.Entity) error      { return nil }
func (m *MockModuleRepo) Update(ctx context.Context, module *models.Entity) error      { return nil }
func (m *MockModuleRepo) Delete(ctx context.Context, name string, userID string) error { return nil }
func (m *MockModuleRepo) List(ctx context.Context) ([]models.Entity, error)            { return nil, nil }
func (m *MockModuleRepo) EnsureGlobalIndexes(ctx context.Context) error                { return nil }
func (m *MockModuleRepo) FindUsingLookup(ctx context.Context, targetModule string) ([]models.Entity, error) {
	return nil, nil
}
func (m *MockModuleRepo) EnsureIndexes(ctx context.Context) error                  { return nil }
func (m *MockModuleRepo) GetDefaults(ctx context.Context) ([]models.Entity, error) { return nil, nil }

type MockRoleService struct {
	Perms map[string]string
}

func (m *MockRoleService) GetFieldPermissions(ctx context.Context, userID primitive.ObjectID, module string) (map[string]string, error) {
	return m.Perms, nil
}
func (m *MockRoleService) GetAccessFilter(ctx context.Context, userID primitive.ObjectID, module string, action string) (primitive.M, error) {
	return nil, nil
}
func (m *MockRoleService) AssignRole(ctx context.Context, userID primitive.ObjectID, roleID string) error {
	return nil
}
func (m *MockRoleService) RemoveRole(ctx context.Context, userID primitive.ObjectID, roleID string) error {
	return nil
}
func (m *MockRoleService) GetUserRoles(ctx context.Context, userID primitive.ObjectID) ([]role.Role, error) {
	return nil, nil
}
func (m *MockRoleService) CreateRole(ctx context.Context, r *role.Role) (*role.Role, error) {
	return nil, nil
}
func (m *MockRoleService) GetRoleByID(ctx context.Context, id string) (*role.Role, error) {
	return nil, nil
}
func (m *MockRoleService) GetRoleByName(ctx context.Context, name string) (*role.Role, error) {
	return nil, nil
}
func (m *MockRoleService) ListRoles(ctx context.Context) ([]role.Role, error)            { return nil, nil }
func (m *MockRoleService) UpdateRole(ctx context.Context, id string, r *role.Role) error { return nil }
func (m *MockRoleService) DeleteRole(ctx context.Context, id string) error               { return nil }
func (m *MockRoleService) GetPermissionsForRoles(ctx context.Context, roleIDHexes []string) ([]string, error) {
	return nil, nil
}
func (m *MockRoleService) CheckModulePermission(ctx context.Context, roleNames []string, moduleName string, permission string) (bool, error) {
	return true, nil
}
func (m *MockRoleService) CheckPermission(ctx context.Context, userID primitive.ObjectID, resourceID string, action string) (bool, error) {
	return true, nil
}

type MockAutomationTrigger struct{}

func (m *MockAutomationTrigger) ExecuteFromTrigger(ctx context.Context, moduleName string, record map[string]any, triggerType string) error {
	return nil
}

type MockWebhookService struct{}

func (m *MockWebhookService) Trigger(ctx context.Context, event string, payload common_models.WebhookPayload) {
}                                                                                         // No return
func (m *MockWebhookService) CreateWebhook(ctx context.Context, w *webhook.Webhook) error { return nil }
func (m *MockWebhookService) ListWebhooks(ctx context.Context) ([]webhook.Webhook, error) {
	return nil, nil
}
func (m *MockWebhookService) GetWebhook(ctx context.Context, id string) (*webhook.Webhook, error) {
	return nil, nil
}
func (m *MockWebhookService) UpdateWebhook(ctx context.Context, id string, updates map[string]any) error {
	return nil
}
func (m *MockWebhookService) DeleteWebhook(ctx context.Context, id string) error { return nil }

func TestUpdateRecord_BlueprintValidation(t *testing.T) {
	// Setup Mocks
	mockRepo := &MockRecordRepo{}
	mockModuleRepo := &MockModuleRepo{
		Entity: &models.Entity{
			Name: "deals",
			Fields: []models.ModuleField{
				{Name: "stage", Label: "Stage", Type: models.FieldTypeText},
				{Name: "amount", Label: "Amount", Type: models.FieldTypeNumber},
			},
		},
	}
	mockRoleService := &MockRoleService{}
	mockValidator := &MockBlueprintValidator{}
	mockAudit := &MockAuditService{} // Defined in soft_delete_test.go
	mockAutomation := &MockAutomationTrigger{}
	mockWebhook := &MockWebhookService{}

	service := &RecordServiceImpl{
		RecordRepo:         mockRepo,
		ModuleRepo:         mockModuleRepo,
		RoleService:        mockRoleService,
		BlueprintValidator: mockValidator,
		AuditService:       mockAudit,
		AutomationService:  mockAutomation,
		WebhookService:     mockWebhook,
	}

	ctx := context.Background()
	userID := primitive.NewObjectID()
	recordID := primitive.NewObjectID().Hex()

	// Case 1: Active Blueprint manages "stage". Update contains "stage". Should Fail.
	t.Run("BlockedByBlueprint", func(t *testing.T) {
		mockValidator.TargetField = "stage"
		data := map[string]any{
			"stage": "Closed Won",
		}

		err := service.UpdateRecord(ctx, "deals", recordID, data, userID)
		if err == nil {
			t.Error("Expected error due to blueprint validation, got nil")
		} else if err.Error() != "field 'stage' is managed by a blueprint and cannot be updated manually" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	// Case 2: Active Blueprint manages "stage". Update only "amount". Should Pass.
	t.Run("AllowedUpdate", func(t *testing.T) {
		mockValidator.TargetField = "stage"
		data := map[string]any{
			"amount": 1000,
		}

		err := service.UpdateRecord(ctx, "deals", recordID, data, userID)
		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}
	})

	// Case 3: No Active Blueprint. Update "stage". Should Pass.
	t.Run("NoBlueprint", func(t *testing.T) {
		mockValidator.TargetField = ""
		data := map[string]any{
			"stage": "Negotiation",
		}

		err := service.UpdateRecord(ctx, "deals", recordID, data, userID)
		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}
	})

	// Case 4: Active Blueprint manages "stage". Update contains "stage" with same value. Should Pass.
	t.Run("SameValueUpdate", func(t *testing.T) {
		mockValidator.TargetField = "stage"
		mockRepo.Record = map[string]any{
			"_id":   recordID,
			"stage": "Negotiation",
		}
		data := map[string]any{
			"stage": "Negotiation",
		}

		err := service.UpdateRecord(ctx, "deals", recordID, data, userID)
		if err != nil {
			t.Errorf("Expected success for same value update, got error: %v", err)
		}
	})
}
