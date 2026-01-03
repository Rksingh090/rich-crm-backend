package permission

import (
	"context"
	"fmt"
	"time"

	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/user"

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
}

type PermissionServiceImpl struct {
	PermissionRepo PermissionRepository
	UserRepo       user.UserRepository
	AuditService   audit.AuditService
}

func NewPermissionService(
	permissionRepo PermissionRepository,
	userRepo user.UserRepository,
	auditService audit.AuditService,
) PermissionService {
	return &PermissionServiceImpl{
		PermissionRepo: permissionRepo,
		UserRepo:       userRepo,
		AuditService:   auditService,
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
		"role_id":  {New: permission.RoleID.Hex()},
		"resource": {New: permission.Resource},
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
		"resource": {Old: perm.Resource},
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
		ID:       primitive.NewObjectID(),
		TenantID: tenantID,
		RoleID:   roleID,
		Resource: ResourceRef{
			Type: "module", // Default, should be determined from resource registry
			ID:   req.ResourceID,
		},
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
	// Get user to find their roles
	user, err := s.UserRepo.FindByID(ctx, userID.Hex())
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Aggregate permissions from all roles
	effectivePerms := make(map[string]*Permission)

	for _, roleID := range user.Roles {
		perms, err := s.PermissionRepo.FindByRoleID(ctx, roleID.Hex())
		if err != nil {
			continue
		}

		for _, perm := range perms {
			resourceKey := perm.Resource.ID

			if existing, ok := effectivePerms[resourceKey]; ok {
				// Merge permissions (most permissive wins)
				for action, actionPerm := range perm.Actions {
					if existingAction, exists := existing.Actions[action]; exists {
						// If either allows without conditions, allow without conditions
						if actionPerm.Allowed && actionPerm.Conditions == nil {
							existing.Actions[action] = actionPerm
						} else if actionPerm.Allowed && !existingAction.Allowed {
							existing.Actions[action] = actionPerm
						}
					} else {
						existing.Actions[action] = actionPerm
					}
				}

				// Merge field rules (most permissive wins)
				for field, rule := range perm.FieldRules {
					if existingRule, exists := existing.FieldRules[field]; exists {
						if rule == "read_write" {
							existing.FieldRules[field] = "read_write"
						} else if rule == "read_only" && existingRule == "none" {
							existing.FieldRules[field] = "read_only"
						}
					} else {
						existing.FieldRules[field] = rule
					}
				}
			} else {
				// First time seeing this resource
				permCopy := perm
				effectivePerms[resourceKey] = &permCopy
			}
		}
	}

	return effectivePerms, nil
}
