package resource

import (
	"context"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/module"
	"go-crm/internal/features/permission"
	"go-crm/internal/features/role"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ResourceService interface {
	ListResources(ctx context.Context) ([]Resource, error)
	ListSidebarResources(ctx context.Context, product string, location string) ([]Resource, error)
	GetSidebar(ctx context.Context, userID string) ([]Resource, error)
	SyncResources(ctx context.Context, resources []Resource) error
	CreateResource(ctx context.Context, resource *Resource) error
	DeleteResource(ctx context.Context, resourceID string, userID string) error
	GetResourceMetadata(ctx context.Context, resourceName string, action string, userID string) (map[string]interface{}, error)
}

type ResourceServiceImpl struct {
	repo              ResourceRepository
	roleService       role.RoleService
	permissionService permission.PermissionService
	moduleRepo        module.ModuleRepository
}

func NewResourceService(repo ResourceRepository, roleService role.RoleService, permissionService permission.PermissionService, moduleRepo module.ModuleRepository) ResourceService {
	return &ResourceServiceImpl{
		repo:              repo,
		roleService:       roleService,
		permissionService: permissionService,
		moduleRepo:        moduleRepo,
	}
}

func (s *ResourceServiceImpl) ListResources(ctx context.Context) ([]Resource, error) {
	return s.repo.FindAll(ctx)
}

func (s *ResourceServiceImpl) SyncResources(ctx context.Context, resources []Resource) error {
	for _, res := range resources {
		// Populate ResourceID if missing (legacy support or new convention)
		if res.ResourceID == "" {
			res.ResourceID = res.Product + "." + res.Key
		}

		// Try to find existing resource by ResourceID (unique string identifier)
		existing, err := s.repo.FindByResourceID(ctx, res.ResourceID)
		if err == nil && existing != nil {
			// Update existing resource
			res.ID = existing.ID             // Keep the existing ObjectID
			res.TenantID = existing.TenantID // Preserve existing tenant_id if set
			res.CreatedAt = existing.CreatedAt
			res.UpdatedAt = time.Now()
			if err := s.repo.Update(ctx, &res); err != nil {
				// If update fails due to tenant mismatch, try direct update
				return err
			}
		} else {
			// Create new resource
			if res.ID.IsZero() {
				res.ID = primitive.NewObjectID()
			}
			res.CreatedAt = time.Now()
			res.UpdatedAt = time.Now()
			if err := s.repo.Create(ctx, &res); err != nil {
				// Ignore duplicate key errors (resource already exists)
				if !strings.Contains(err.Error(), "duplicate key") {
					return err
				}
			}
		}
	}
	return nil
}

func (s *ResourceServiceImpl) GetSidebar(ctx context.Context, userID string) ([]Resource, error) {
	// 1. Fetch all resources
	allResources, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	// 2. Filter by Sidebar = true and Permissions
	var sidebarResources []Resource

	// Convert string userID to ObjectID
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	for _, res := range allResources {
		if !res.UI.Sidebar {
			continue
		}

		// Check Permission
		// Assuming RoleService has a CheckPermission method that takes userID
		// We will need to implement/expose this.
		// For now, using a placeholder logic or assuming the method exists.
		allowed, err := s.roleService.CheckPermission(ctx, uID, res.ResourceID, "read")
		if err != nil {
			continue
		}

		if allowed {
			sidebarResources = append(sidebarResources, res)
		}
	}

	return sidebarResources, nil
}

func (s *ResourceServiceImpl) ListSidebarResources(ctx context.Context, product string, location string) ([]Resource, error) {
	return s.repo.FindSidebarResources(ctx, product, location)
}

func (s *ResourceServiceImpl) CreateResource(ctx context.Context, resource *Resource) error {
	if resource.ID.IsZero() {
		resource.ID = primitive.NewObjectID()
	}
	resource.CreatedAt = time.Now()
	resource.UpdatedAt = time.Now()
	return s.repo.Create(ctx, resource)
}

func (s *ResourceServiceImpl) DeleteResource(ctx context.Context, resourceID string, userID string) error {
	// Find resource by ResourceID (e.g., "crm.leads")
	resource, err := s.repo.FindByResourceID(ctx, resourceID)
	if err != nil {
		return err
	}

	// Prevent deletion of system resources
	if resource.IsSystem {
		return fmt.Errorf("cannot delete system resource")
	}

	return s.repo.Delete(ctx, resource.ID.Hex(), userID)
}

func (s *ResourceServiceImpl) GetResourceMetadata(ctx context.Context, resourceName string, action string, userID string) (map[string]interface{}, error) {
	// 1. Fetch Resource Schema (Entity) from Module Repository
	// ResourceName in API (e.g. "crm.leads") should match Module Name
	moduleEntity, err := s.moduleRepo.FindByName(ctx, resourceName)
	if err != nil {
		return nil, fmt.Errorf("resource schema not found: %v", err)
	}

	// Also fetch Resource definition for Label if needed, or use Module Label
	// moduleEntity has Label.

	// 2. Fetch User Effective Permissions
	oid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}

	perms, err := s.permissionService.GetUserEffectivePermissions(ctx, oid)
	if err != nil {
		return nil, fmt.Errorf("failed to load permissions: %v", err)
	}

	// 3. Check Action Permission
	var actionPerm *common_models.ActionPermission

	// Check specific resource permission
	if p, ok := perms[resourceName]; ok {
		if ap, ok := p.Actions[action]; ok {
			actionPerm = &ap
		}
	}

	// Fallback to wildcard if not found (or should we merge? EffectivePermissions should handle this?)
	// GetUserEffectivePermissions handles merging of * and resourceID permissions?
	// Waiting: implementation of GetUserEffectivePermissions merges based on "most permissive wins".
	// But it returns a map[string]*Permission. It separates "*" and "resourceID".
	// We need to check both here if "EffectivePermissions" didn't already merge wildcard into specific.
	// Looking at PermissionService implementation:
	// It iterates perms. "if existing, ok := effectivePerms[resourceKey]".
	// It does NOT merge "wildcard" into "specific resources" automatically unless the loop encounters them.
	// So effectivePerms["*"] is separate from effectivePerms["crm.leads"].

	if actionPerm == nil {
		// Check wildcard
		if p, ok := perms["*"]; ok {
			if ap, ok := p.Actions[action]; ok {
				actionPerm = &ap
			}
		}
	}

	allowed := false
	if actionPerm != nil {
		allowed = actionPerm.Allowed
	}

	// 4. Derive Filters
	var allowedFilters []map[string]interface{}

	if allowed {
		// Calculate available filters from Schema
		schemaFilters := make(map[string]common_models.ModuleField)
		for _, f := range moduleEntity.Fields {
			if f.Filterable {
				schemaFilters[f.Name] = f
			}
		}

		// Apply Permission Restrictions
		// If ui.filters is defined, intersect. If nil/empty, NO filters allowed?
		// Requirement: "If permission does not define ui.filters, show none."
		// Note: ActionPermission.UI might be nil.

		var permFilters []string
		if actionPerm.UI != nil {
			permFilters = actionPerm.UI.Filters
		}

		if len(permFilters) > 0 {
			for _, key := range permFilters {
				if field, ok := schemaFilters[key]; ok {
					allowedFilters = append(allowedFilters, map[string]interface{}{
						"key":     field.Name,
						"label":   field.Label,
						"type":    field.Type,
						"options": field.Options,
					})
				}
			}
		}
	}

	response := map[string]interface{}{
		"resource": moduleEntity.Name, // Or ID if that's what we want
		"label":    moduleEntity.Label,
		"actions": map[string]interface{}{
			action: map[string]interface{}{
				"allowed": allowed,
				"filters": allowedFilters,
			},
		},
	}

	return response, nil
}

// Helper to convert resource.ModuleField to common_models.ModuleField if they differ,
// currently they seem to be the same struct or similar structure.
// Wait, Resource struct in resource/model.go has fields.
// Resource struct uses `ModuleField`?
// Checking resource/model.go... it seems it defines Resource but not nested Field struct, wait.
// In read file 12, Resource struct has `Fields` but field struct was not shown?
// Ah, implicit or imported? file 12 did not import common `models`.
// Let's check file 12 again.
// File 12: `Fields []string`? No.
// Line 17: Type Resource struct...
// I missed checking where `ModuleField` is defined for `Resource`.
// File 12 does NOT show `Fields` in `Resource` struct!
// Wait.
// Line 27: `Actions []string`
// Line 33: `UI ResourceUI`
// Where are the fields?
// Ah, `Resource` (Entity) definition in USER PROMPT 1.1 shows `fields`.
// But the GO MODEL in file 12 might NOT have it yet or I missed it.
// Let's re-read file 12 content carefully (step 12).
// It ends at line 40. Struct Resource starts at 17.
// Fields are NOT in `Resource` struct in `resource/model.go`!
// Wait. `ModuleRepository` returns `Resource`.
// `Feature/module` deals with schema?
// `Resource` might be just for navigation/sidebar?
// But prompt says "1.1 entities (module_schema) ... Resource ... defines module structure".
// Maybe I need to update `Resource` struct to include `Fields`.
// Or maybe `Module` struct in `feature/module` is the one holding fields?
// Confusion: `Resource` vs `Module`.
// Prompt 1.1 says: `"_id": "crm.leads", "type": "module"`.
// Prompt 1.3 says: `"resource": "crm.leads"`.
// In `record/service.go`, `m, err := s.ModuleRepo.FindByName(ctx, moduleName)`.
// `record` service uses `ModuleRepo`.
// `resource` service uses `ResourceRepo`.
// Are they the same?
// `ModuleRepo` returns `m` which has `Fields`.
// `ResourceRepo` returns `Resource` (which lacks fields in file 12).
// I suspect `Resource` entity in `resource` feature IS the metadata, but `Module` feature holds the schema details?
// OR, `Resource` needs to be updated to include Fields as per "Update Resource (Entity) model" task.
// YES. Task 1: "Update Resource (Entity) model to match new schema structure".
// So I MUST add `Fields` to `Resource` struct in `resource/model.go`.
// AND probably switch `resource/service.go` to use that.
// BUT `record/service.go` uses `module.ModuleRepository`.
// Is `resource` replacing `module`?
// The user prompt calls "1.1 entities (module_schema)".
// I should probably unify or ensure `Resource` has fields.
// Let's check `backend/internal/features/module/model.go`.
