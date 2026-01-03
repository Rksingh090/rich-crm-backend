package role

import (
	"context"
	"fmt"
	"time"

	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/permission"
	"go-crm/internal/features/user"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RoleService interface {
	CreateRole(ctx context.Context, role *Role) (*Role, error)
	GetRoleByID(ctx context.Context, id string) (*Role, error)
	GetRoleByName(ctx context.Context, name string) (*Role, error)
	ListRoles(ctx context.Context) ([]Role, error)
	UpdateRole(ctx context.Context, id string, role *Role) error
	DeleteRole(ctx context.Context, id string) error
	GetPermissionsForRoles(ctx context.Context, roleIDHexes []string) ([]string, error)
	CheckModulePermission(ctx context.Context, roleNames []string, moduleName string, permission string) (bool, error)
	GetFieldPermissions(ctx context.Context, userID primitive.ObjectID, moduleName string) (map[string]string, error)
	GetAccessFilter(ctx context.Context, userID primitive.ObjectID, moduleName string, action string) (bson.M, error)
	CheckPermission(ctx context.Context, userID primitive.ObjectID, resourceID string, action string) (bool, error)
}

type RoleServiceImpl struct {
	RoleRepo          RoleRepository
	UserRepo          user.UserRepository
	AuditService      audit.AuditService
	PermissionService permission.PermissionService
}

func NewRoleService(
	roleRepo RoleRepository,
	userRepo user.UserRepository,
	auditService audit.AuditService,
	permissionService permission.PermissionService,
) RoleService {
	return &RoleServiceImpl{
		RoleRepo:          roleRepo,
		UserRepo:          userRepo,
		AuditService:      auditService,
		PermissionService: permissionService,
	}
}

func (s *RoleServiceImpl) CreateRole(ctx context.Context, role *Role) (*Role, error) {
	role.ID = primitive.NewObjectID()
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()

	if role.Permissions == nil {
		role.Permissions = make(map[string]map[string]common_models.ActionPermission)
	}

	if err := s.RoleRepo.Create(ctx, role); err != nil {
		return nil, err
	}

	_ = s.AuditService.LogChange(ctx, common_models.AuditActionCreate, "role", role.ID.Hex(), map[string]common_models.Change{
		"name": {New: role.Name},
	})

	return role, nil
}

func (s *RoleServiceImpl) GetRoleByID(ctx context.Context, id string) (*Role, error) {
	return s.RoleRepo.FindByID(ctx, id)
}

func (s *RoleServiceImpl) GetRoleByName(ctx context.Context, name string) (*Role, error) {
	return s.RoleRepo.FindByName(ctx, name)
}

func (s *RoleServiceImpl) ListRoles(ctx context.Context) ([]Role, error) {
	return s.RoleRepo.List(ctx)
}

func (s *RoleServiceImpl) UpdateRole(ctx context.Context, id string, role *Role) error {
	role.UpdatedAt = time.Now()

	if err := s.RoleRepo.Update(ctx, id, role); err != nil {
		return err
	}

	_ = s.AuditService.LogChange(ctx, common_models.AuditActionUpdate, "role", id, map[string]common_models.Change{
		"permissions": {New: role.Permissions},
	})

	return nil
}

// ... DeleteRole ...

func (s *RoleServiceImpl) DeleteRole(ctx context.Context, id string) error {
	// Check if role is a system role
	role, err := s.RoleRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if role.IsSystem {
		return fmt.Errorf("cannot delete system role")
	}

	if err := s.RoleRepo.Delete(ctx, id); err != nil {
		return err
	}

	_ = s.AuditService.LogChange(ctx, common_models.AuditActionDelete, "role", id, map[string]common_models.Change{
		"name": {Old: role.Name},
	})

	return nil
}

func (s *RoleServiceImpl) GetPermissionsForRoles(ctx context.Context, roleIDHexes []string) ([]string, error) {
	var roleIDs []interface{}
	for _, rID := range roleIDHexes {
		oid, err := primitive.ObjectIDFromHex(rID)
		if err == nil {
			roleIDs = append(roleIDs, oid)
		}
	}

	if len(roleIDs) == 0 {
		return []string{}, nil
	}

	return s.RoleRepo.FindPermissionsByRoleIDs(ctx, roleIDs)
}

func (s *RoleServiceImpl) CheckModulePermission(ctx context.Context, roleNames []string, moduleName string, permission string) (bool, error) {
	// Legacy method relied on roleNames, but new system relies on UserID for effective permissions
	// Extract userID from context if available (AuthMiddleware usually puts "user_id" in Locals, need fiber context?)
	// But this is service layer, relying on context values passed from controller/middleware.
	// The standard way: Middleware should pass UserID or we extract it from context if stored there.
	// If not available, we fall back to checking roles individually via PermissionService (less efficient/accurate for ABAC).

	// Better approach: Use the passed roleNames to find roles, then check permissions for those roles.
	// But our new system aggregates permissions.

	// Helper to check a single role's permission via new system
	checkRole := func(roleName string) (bool, error) {
		role, err := s.RoleRepo.FindByName(ctx, roleName)
		if err != nil {
			return false, err
		}
		perms, err := s.PermissionService.GetPermissionsByRole(ctx, role.ID.Hex())
		if err != nil {
			return false, err
		}

		resourceID := "crm." + moduleName // Assumption mapping
		if moduleName == "*" {
			resourceID = "*"
		}

		for _, p := range perms {
			if p.Resource.ID == resourceID || p.Resource.ID == "*" {
				if actionPerm, ok := p.Actions[permission]; ok && actionPerm.Allowed {
					return true, nil
				}
			}
		}
		return false, nil
	}

	for _, name := range roleNames {
		// Check for Super Admin bypass in role name
		if name == "Super Admin" || name == "admin" {
			return true, nil
		}
		allowed, err := checkRole(name)
		if err == nil && allowed {
			return true, nil
		}
	}

	return false, nil
}

// ... GetFieldPermissions ...

func (s *RoleServiceImpl) GetFieldPermissions(ctx context.Context, userID primitive.ObjectID, moduleName string) (map[string]string, error) {
	// ... implementation same as before but referring to FieldPermissions which is separate ...
	// Checking previous code: FieldPermissions is a separate map on Role.
	// So this method likely doesn't check ModulePermissions/Permissions.
	// Let's verify existing code content.
	// Existing code just accesses Role.FieldPermissions. This is fine.
	// I will just copy it or leave it if not touched.
	// Wait, I am replacing the whole block, so I need to include it.

	// 1. Get User
	user, err := s.UserRepo.FindByID(ctx, userID.Hex())
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// 2. Iterate Roles
	finalPerms := make(map[string]string)
	hasFieldRules := false

	for _, roleID := range user.Roles {
		role, err := s.RoleRepo.FindByID(ctx, roleID.Hex())
		if err != nil || role == nil {
			continue
		}

		if role.FieldPermissions != nil {
			if modPerms, ok := role.FieldPermissions[moduleName]; ok {
				hasFieldRules = true
				for field, p := range modPerms {
					current, exists := finalPerms[field]
					if !exists {
						finalPerms[field] = p
					} else {
						switch p {
						case FieldPermReadWrite:
							finalPerms[field] = FieldPermReadWrite
						case FieldPermReadOnly:
							if current == FieldPermNone {
								finalPerms[field] = FieldPermReadOnly
							}
						}
					}
				}
			}
		}

		if role.FieldPermissions == nil || role.FieldPermissions[moduleName] == nil {
			return nil, nil // Full access
		}
	}

	if !hasFieldRules {
		return nil, nil
	}

	return finalPerms, nil
}

func (s *RoleServiceImpl) GetAccessFilter(ctx context.Context, userID primitive.ObjectID, moduleName string, action string) (primitive.M, error) {
	// 1. Get User
	user, err := s.UserRepo.FindByID(ctx, userID.Hex())
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	var orConditions []primitive.M
	hasFullAccess := false

	// Prepared Context Data for Variables
	orgID := user.TenantID
	userGroups := user.Groups
	if userGroups == nil {
		userGroups = []string{}
	}
	contextData := PrepareContextData(userID, orgID, userGroups)

	for _, roleID := range user.Roles {
		// Check Admin Bypass (Optional, but safe)
		role, err := s.RoleRepo.FindByID(ctx, roleID.Hex())
		if err == nil && (role.Name == "admin" || role.Name == "Super Admin") {
			return primitive.M{}, nil // Full Access
		}

		// Fetch Permissions from Service (Source of Truth)
		perms, err := s.PermissionService.GetPermissionsByRole(ctx, roleID.Hex())
		if err != nil {
			continue
		}

		for _, p := range perms {
			// Check Wildcard or Specific Resource
			if p.Resource.ID == "*" || p.Resource.ID == moduleName {
				if actionPerm, ok := p.Actions[action]; ok && actionPerm.Allowed {
					if actionPerm.Conditions == nil {
						hasFullAccess = true
					} else {
						cond, err := TranslateConditions(actionPerm.Conditions, contextData)
						if err == nil {
							orConditions = append(orConditions, cond)
						}
					}
				}
			}
		}
	}

	if hasFullAccess {
		return primitive.M{}, nil
	}

	if len(orConditions) == 0 {
		return primitive.M{"_id": -1}, nil
	}

	if len(orConditions) == 1 {
		return orConditions[0], nil
	}

	return primitive.M{"$or": orConditions}, nil
}

func (s *RoleServiceImpl) CheckPermission(ctx context.Context, userID primitive.ObjectID, resourceID string, action string) (bool, error) {
	// Use PermissionService to get effective permissions (RBAC/ABAC source of truth)
	effectivePerms, err := s.PermissionService.GetUserEffectivePermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	// 1. Check Wildcard Resource "*"
	if wildPerm, ok := effectivePerms["*"]; ok {
		if p, ok := wildPerm.Actions[action]; ok && p.Allowed {
			return true, nil
		}
	}

	// 2. Check Specific Resource
	if resPerm, ok := effectivePerms[resourceID]; ok {
		if p, ok := resPerm.Actions[action]; ok && p.Allowed {
			return true, nil
		}
	}

	return false, nil
}
