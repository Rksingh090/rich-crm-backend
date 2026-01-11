package permission

import (
	"context"
	"fmt"
	"time"

	common_models "go-crm/internal/common/models"
	"go-crm/internal/core/audit"
	"go-crm/internal/core/organization"
	"go-crm/internal/core/user"
	"go-crm/internal/features/group"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PermissionService interface {
	CreatePermission(ctx context.Context, permission *Permission) (*Permission, error)
	GetPermissionByID(ctx context.Context, id string) (*Permission, error)
	GetPermissionsByRole(ctx context.Context, roleID string) ([]Permission, error)
	GetPermissionsByResource(ctx context.Context, resourceType, resourceID string) ([]Permission, error)
	UpdatePermission(ctx context.Context, id string, permission *Permission) error
	DeletePermission(ctx context.Context, id string) error
	AssignResourceToRole(ctx context.Context, req AssignResourceRequest) error
	RevokeResourceFromRole(ctx context.Context, req RevokeResourceRequest) error
	GetUserEffectivePermissions(ctx context.Context, userID primitive.ObjectID) (map[string]*Permission, error)
	InspectPermissions(ctx context.Context, userID primitive.ObjectID, resourceID string) (*InspectionResult, error)
}

type InspectionStep struct {
	Layer   string `json:"layer"`
	Source  string `json:"source"`
	Details string `json:"details"`
}

type InspectionResult struct {
	Effective *Permission      `json:"effective"`
	Trace     []InspectionStep `json:"trace"`
}

type PermissionServiceImpl struct {
	PermissionRepo   PermissionRepository
	UserRepo         user.UserRepository
	GroupRepo        group.GroupRepository
	OrganizationRepo organization.OrganizationRepository
	AuditService     audit.AuditService
}

func NewPermissionService(
	permissionRepo PermissionRepository,
	userRepo user.UserRepository,
	groupRepo group.GroupRepository,
	orgRepo organization.OrganizationRepository,
	auditService audit.AuditService,
) PermissionService {
	return &PermissionServiceImpl{
		PermissionRepo:   permissionRepo,
		UserRepo:         userRepo,
		GroupRepo:        groupRepo,
		OrganizationRepo: orgRepo,
		AuditService:     auditService,
	}
}

func (s *PermissionServiceImpl) CreatePermission(ctx context.Context, permission *Permission) (*Permission, error) {
	permission.ID = primitive.NewObjectID()
	permission.CreatedAt = time.Now()
	permission.UpdatedAt = time.Now()

	if permission.Actions == nil {
		permission.Actions = make(map[string]common_models.ActionPermission)
	}

	if err := s.PermissionRepo.Create(ctx, permission); err != nil {
		return nil, err
	}

	_ = s.AuditService.LogChange(ctx, common_models.AuditActionCreate, "permission", permission.ID.Hex(), map[string]common_models.Change{
		"role_id":     {New: permission.RoleID.Hex()},
		"resource_id": {New: permission.ResourceID},
	})

	return permission, nil
}

func (s *PermissionServiceImpl) GetPermissionByID(ctx context.Context, id string) (*Permission, error) {
	return s.PermissionRepo.FindByID(ctx, id)
}

func (s *PermissionServiceImpl) GetPermissionsByRole(ctx context.Context, roleID string) ([]Permission, error) {
	return s.PermissionRepo.FindByRoleID(ctx, roleID)
}

func (s *PermissionServiceImpl) GetPermissionsByResource(ctx context.Context, resourceType, resourceID string) ([]Permission, error) {
	return s.PermissionRepo.FindByResource(ctx, resourceType, resourceID)
}

func (s *PermissionServiceImpl) UpdatePermission(ctx context.Context, id string, permission *Permission) error {
	permission.UpdatedAt = time.Now()

	if err := s.PermissionRepo.Update(ctx, id, permission); err != nil {
		return err
	}

	_ = s.AuditService.LogChange(ctx, common_models.AuditActionUpdate, "permission", id, map[string]common_models.Change{
		"actions": {New: permission.Actions},
	})

	return nil
}

func (s *PermissionServiceImpl) DeletePermission(ctx context.Context, id string) error {
	perm, err := s.PermissionRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.PermissionRepo.Delete(ctx, id); err != nil {
		return err
	}

	_ = s.AuditService.LogChange(ctx, common_models.AuditActionDelete, "permission", id, map[string]common_models.Change{
		"resource_id": {Old: perm.ResourceID},
	})

	return nil
}

func (s *PermissionServiceImpl) AssignResourceToRole(ctx context.Context, req AssignResourceRequest) error {
	roleID, err := primitive.ObjectIDFromHex(req.RoleID)
	if err != nil {
		return fmt.Errorf("invalid role ID: %v", err)
	}

	// Get tenant ID from context
	tenantIDStr, ok := ctx.Value(common_models.TenantIDKey).(string)
	if !ok {
		return fmt.Errorf("tenant ID not found in context")
	}
	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		return fmt.Errorf("invalid tenant ID: %v", err)
	}

	// Check if permission already exists
	existing, err := s.PermissionRepo.FindByRoleAndResource(ctx, req.RoleID, req.ResourceID)
	if err != nil {
		return err
	}

	if existing != nil {
		// Update existing permission
		existing.Actions = req.Actions
		existing.FieldRules = req.FieldRules
		existing.UpdatedAt = time.Now()
		return s.PermissionRepo.Update(ctx, existing.ID.Hex(), existing)
	}

	// Create new permission
	permission := &Permission{
		ID:         primitive.NewObjectID(),
		TenantID:   tenantID,
		RoleID:     roleID,
		ResourceID: req.ResourceID, // Reference to Resource Registry
		Actions:    req.Actions,
		FieldRules: req.FieldRules,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	return s.PermissionRepo.Create(ctx, permission)
}

func (s *PermissionServiceImpl) RevokeResourceFromRole(ctx context.Context, req RevokeResourceRequest) error {
	existing, err := s.PermissionRepo.FindByRoleAndResource(ctx, req.RoleID, req.ResourceID)
	if err != nil {
		return err
	}

	if existing == nil {
		return fmt.Errorf("permission not found")
	}

	return s.PermissionRepo.Delete(ctx, existing.ID.Hex())
}

func (s *PermissionServiceImpl) GetUserEffectivePermissions(ctx context.Context, userID primitive.ObjectID) (map[string]*Permission, error) {
	// 1. Get User
	user, err := s.UserRepo.FindByID(ctx, userID.Hex())
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// 1b. Check Platform Admin (Bypass)
	if user.IsPlatformAdmin {
		return map[string]*Permission{
			"*": {
				ResourceID: "*",
				Actions: map[string]common_models.ActionPermission{
					"read":   {Allowed: true},
					"create": {Allowed: true},
					"update": {Allowed: true},
					"delete": {Allowed: true},
				},
				FieldRules: map[string]string{}, // Full access implicitly
			},
		}, nil
	}

	// Initialize effective permissions map
	effectivePerms := make(map[string]*Permission)

	// Helper to merge permissions
	// Priority: The caller controls priority by calling merge in order (Lowest to Highest)
	// Strategy: "Specific Overrides General". If a higher layer defines a permission (Allowed=true/false), it overwrites.
	// However, for multiple items at SAME level (e.g. multiple roles), we usually Union (Allow wins).
	// Current Implementation: Sequential overwrite.
	merge := func(resourceID string, actions map[string]common_models.ActionPermission, fieldRules map[string]string) {
		existing, ok := effectivePerms[resourceID]
		if !ok {
			existing = &Permission{
				ResourceID: resourceID,
				Actions:    make(map[string]common_models.ActionPermission),
				FieldRules: make(map[string]string),
			}
			effectivePerms[resourceID] = existing
		}

		// Merge Actions
		for action, perm := range actions {
			// Logic: If existing is set, we overwrite ONLY if this layer is higher priority?
			// But this helper is called for each layer.
			// Problem: How to handle "Union" within layer (Role A vs Role B) vs "Override" between layers (User vs Role).
			// This helper assumes it is called in order of priority.
			// But for Roles, we need Union.
			// So Roles should be merged FIRST into a temporary map, then that map merged here?
			// Let's simplify:
			// If we process layers from Lowest (Org) to Highest (User).
			// Then later overwrites earlier.
			existing.Actions[action] = perm
		}

		// Merge Field Rules
		for field, rule := range fieldRules {
			existing.FieldRules[field] = rule
		}
	}

	// Helper helper for union merging (within same level)
	mergeUnion := func(target *Permission, resourceID string, actions map[string]common_models.ActionPermission, fieldRules map[string]string) {
		if target.ResourceID == "" {
			target.ResourceID = resourceID
			target.Actions = make(map[string]common_models.ActionPermission)
			target.FieldRules = make(map[string]string)
		}

		for action, perm := range actions {
			if existingPerm, ok := target.Actions[action]; ok {
				// Union: If either allows, allow.
				if perm.Allowed || existingPerm.Allowed {
					target.Actions[action] = common_models.ActionPermission{Allowed: true}
					// Conditions merging is complex (OR). skipping for brevity, assuming full Access wins.
				}
			} else {
				target.Actions[action] = perm
			}
		}
		for field, rule := range fieldRules {
			// Union: Allow most permissive?
			// read_write > read_only > none
			existing := target.FieldRules[field]
			if rule == "read_write" || existing == "read_write" {
				target.FieldRules[field] = "read_write"
			} else if rule == "read_only" || existing == "read_only" {
				target.FieldRules[field] = "read_only"
			} else {
				target.FieldRules[field] = rule
			}
		}
	}

	// Layer 4: Organization Defaults
	org, err := s.OrganizationRepo.FindByID(ctx, user.TenantID.Hex())
	if err == nil && org != nil {
		for resID, actions := range org.DefaultPermissions {
			// Extract field rules for this resource
			var rules map[string]string
			if org.DefaultFieldPermissions != nil {
				rules = org.DefaultFieldPermissions[resID]
			}
			merge(resID, actions, rules)
		}
	}

	// Layer 3: Roles (Union of all roles)
	rolePermsMap := make(map[string]*Permission)
	for _, appRole := range user.AppRoles {
		perms, err := s.PermissionRepo.FindByRoleID(ctx, appRole.RoleID.Hex())
		if err != nil {
			continue
		}
		for _, p := range perms {
			resID := p.ResourceID
			if _, ok := rolePermsMap[resID]; !ok {
				rolePermsMap[resID] = &Permission{}
			}
			mergeUnion(rolePermsMap[resID], resID, p.Actions, p.FieldRules)
		}
	}

	// Apply Role Perms (Overwrite Org Defaults)
	for resID, p := range rolePermsMap {
		merge(resID, p.Actions, p.FieldRules)
	}

	// Layer 2: Groups (Union of all groups)
	// Groups are stored in User.Groups as ObjectIDs
	groupPermsMap := make(map[string]*Permission)
	for _, groupID := range user.Groups {
		g, err := s.GroupRepo.FindByID(ctx, groupID)
		if err != nil {
			continue
		}
		if g == nil {
			continue
		}

		for resID, actions := range g.Permissions {
			var rules map[string]string
			if g.FieldPermissions != nil {
				rules = g.FieldPermissions[resID]
			}

			if _, ok := groupPermsMap[resID]; !ok {
				groupPermsMap[resID] = &Permission{}
			}
			mergeUnion(groupPermsMap[resID], resID, actions, rules)
		}
	}
	// Apply Group Perms (Overwrite Roles)
	for resID, p := range groupPermsMap {
		merge(resID, p.Actions, p.FieldRules)
	}

	// Layer 1: User Direct (Overwrite Groups)
	if user.Permissions != nil {
		for resID, actions := range user.Permissions {
			var rules map[string]string
			if user.FieldPermissions != nil {
				rules = user.FieldPermissions[resID]
			}
			merge(resID, actions, rules)
		}
	}

	return effectivePerms, nil
}

func (s *PermissionServiceImpl) InspectPermissions(ctx context.Context, userID primitive.ObjectID, targetResourceID string) (*InspectionResult, error) {
	// Re-implements logic but for a SINGLE resource with tracing

	result := &InspectionResult{
		Trace: []InspectionStep{},
	}

	// 1. Get User
	user, err := s.UserRepo.FindByID(ctx, userID.Hex())
	if err != nil {
		return nil, err
	}

	// Helper to snapshot
	currentPerm := &Permission{
		ResourceID: targetResourceID,
		Actions:    make(map[string]common_models.ActionPermission),
		FieldRules: make(map[string]string),
	}

	logStep := func(layer, source, details string) {
		result.Trace = append(result.Trace, InspectionStep{
			Layer:   layer,
			Source:  source,
			Details: details,
		})
	}

	// Layer 4: Org
	org, err := s.OrganizationRepo.FindByID(ctx, user.TenantID.Hex())
	if err == nil && org != nil {
		if actions, ok := org.DefaultPermissions[targetResourceID]; ok {
			for k, v := range actions {
				currentPerm.Actions[k] = v
			}
			logStep("Organization", "Default", fmt.Sprintf("Found default actions: %v", len(actions)))
		}
		// ... Field rules ...
	}

	// Layer 3: Roles
	for _, appRole := range user.AppRoles {
		roleID := appRole.RoleID
		perms, err := s.PermissionRepo.FindByRoleID(ctx, roleID.Hex())
		if err == nil {
			for _, p := range perms {
				if p.ResourceID == targetResourceID || p.ResourceID == "*" {
					// Union logic simplified for trace
					for k, v := range p.Actions {
						currentPerm.Actions[k] = v
					}
					logStep("Role", roleID.Hex(), fmt.Sprintf("Merged role permissions for %s", p.ResourceID))
				}
			}
		}
	}

	// Layer 2: Groups
	for _, groupID := range user.Groups {
		g, err := s.GroupRepo.FindByID(ctx, groupID)
		if err == nil && g != nil {
			if actions, ok := g.Permissions[targetResourceID]; ok {
				for k, v := range actions {
					currentPerm.Actions[k] = v
				}
				logStep("Group", g.Name, "Overwrote/Merged permissions")
			}
		}
	}

	// Layer 1: User
	if user.Permissions != nil {
		if actions, ok := user.Permissions[targetResourceID]; ok {
			for k, v := range actions {
				currentPerm.Actions[k] = v
			}
			logStep("User", user.Email, "Direct override applied")
		}
	}

	result.Effective = currentPerm
	return result, nil
}
