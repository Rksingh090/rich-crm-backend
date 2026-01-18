package search

import (
	"context"
	"testing"

	"go-crm/internal/common/models"
	"go-crm/internal/core/role"
	"go-crm/internal/features/module"
	"go-crm/internal/features/resource"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockResourceService
type mockResourceService struct {
	resource.ResourceService
	resources []models.Resource
}

func (m *mockResourceService) ListResources(ctx context.Context) ([]models.Resource, error) {
	return m.resources, nil
}

// MockRoleService
type mockRoleService struct {
	role.RoleService
}

func (m *mockRoleService) CheckPermission(ctx context.Context, userID primitive.ObjectID, resourceID string, action string) (bool, error) {
	return true, nil // Allow all for test
}

// MockModuleService
type mockModuleService struct {
	module.ModuleService
}

func (m *mockModuleService) GetModuleByName(ctx context.Context, name string, userID primitive.ObjectID) (*models.Entity, error) {
	return &models.Entity{
		Name:  name,
		Label: name,
		Fields: []models.ModuleField{
			{Name: "name", Type: models.FieldTypeText},
		},
	}, nil
}

func TestGlobalSearch_ResourceMapping(t *testing.T) {
	mockRes := &mockResourceService{
		resources: []models.Resource{
			{
				ResourceID: "crm.leads",
				Type:       "module",
				Key:        "leads",
				Label:      "Leads",
				Route:      "/dashboard/modules/leads",
				Icon:       "Users",
			},
			{
				ResourceID: "crm.settings_general",
				Type:       "setting",
				Key:        "general",
				Label:      "General Settings",
				Route:      "/settings/general",
				Icon:       "Settings",
				UI: models.ResourceUI{
					Location: "settings",
				},
			},
		},
	}

	mockRole := &mockRoleService{}
	mockMod := &mockModuleService{}

	service := &SearchServiceImpl{
		resourceService: mockRes,
		roleService:     mockRole,
		moduleService:   mockMod,
		// db is nil, so we should avoid triggering record search
	}

	ctx := context.Background()
	userID := primitive.NewObjectID()

	t.Run("Match Module", func(t *testing.T) {
		results, err := service.GlobalSearch(ctx, "Le", userID) // Length 2 avoids record search
		if err != nil {
			t.Fatalf("GlobalSearch failed: %v", err)
		}

		if len(results) == 0 {
			t.Fatal("Expected results, got none")
		}

		found := false
		for _, r := range results {
			if r.Type == "module" && r.Title == "Leads" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Module 'Leads' not found in results or type mismatch")
		}
	})

	t.Run("Match Setting", func(t *testing.T) {
		results, err := service.GlobalSearch(ctx, "Ge", userID) // Length 2 avoids record search
		if err != nil {
			t.Fatalf("GlobalSearch failed: %v", err)
		}

		found := false
		for _, r := range results {
			if r.Type == "page" && r.Title == "General Settings" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Setting 'General Settings' not found in results or type mismatch")
		}
	})
}
