package service

import (
	"context"
	"fmt"
	"time"

	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RoleService interface {
	CreateRole(ctx context.Context, role *models.Role) (*models.Role, error)
	GetRoleByID(ctx context.Context, id string) (*models.Role, error)
	GetRoleByName(ctx context.Context, name string) (*models.Role, error)
	ListRoles(ctx context.Context) ([]models.Role, error)
	UpdateRole(ctx context.Context, id string, role *models.Role) error
	DeleteRole(ctx context.Context, id string) error
	GetPermissionsForRoles(ctx context.Context, roleIDHexes []string) ([]string, error)
	CheckModulePermission(ctx context.Context, roleIDs []string, moduleName string, permission string) (bool, error)
	GetFieldPermissions(ctx context.Context, userID primitive.ObjectID, moduleName string) (map[string]string, error)
}

type RoleServiceImpl struct {
	RoleRepo     repository.RoleRepository
	UserRepo     repository.UserRepository
	AuditService AuditService
}

func NewRoleService(roleRepo repository.RoleRepository, userRepo repository.UserRepository, auditService AuditService) RoleService {
	return &RoleServiceImpl{
		RoleRepo:     roleRepo,
		UserRepo:     userRepo,
		AuditService: auditService,
	}
}

func (s *RoleServiceImpl) CreateRole(ctx context.Context, role *models.Role) (*models.Role, error) {
	role.ID = primitive.NewObjectID()
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()

	if role.ModulePermissions == nil {
		role.ModulePermissions = make(map[string]models.ModulePermission)
	}

	if err := s.RoleRepo.Create(ctx, role); err != nil {
		return nil, err
	}

	_ = s.AuditService.LogChange(ctx, models.AuditActionCreate, "role", role.ID.Hex(), map[string]models.Change{
		"name": {New: role.Name},
	})

	return role, nil
}

func (s *RoleServiceImpl) GetRoleByID(ctx context.Context, id string) (*models.Role, error) {
	return s.RoleRepo.FindByID(ctx, id)
}

func (s *RoleServiceImpl) GetRoleByName(ctx context.Context, name string) (*models.Role, error) {
	return s.RoleRepo.FindByName(ctx, name)
}

func (s *RoleServiceImpl) ListRoles(ctx context.Context) ([]models.Role, error) {
	return s.RoleRepo.List(ctx)
}

func (s *RoleServiceImpl) UpdateRole(ctx context.Context, id string, role *models.Role) error {
	role.UpdatedAt = time.Now()

	if err := s.RoleRepo.Update(ctx, id, role); err != nil {
		return err
	}

	_ = s.AuditService.LogChange(ctx, models.AuditActionUpdate, "role", id, map[string]models.Change{
		"permissions": {New: role.ModulePermissions},
	})

	return nil
}

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

	_ = s.AuditService.LogChange(ctx, models.AuditActionDelete, "role", id, map[string]models.Change{
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
	for _, roleName := range roleNames {
		role, err := s.RoleRepo.FindByName(ctx, roleName)
		if err != nil {
			continue
		}

		// Admin Bypass
		if role.Name == "admin" {
			return true, nil
		}

		// Check for wildcard permission first
		if wildcardPerm, exists := role.ModulePermissions["*"]; exists {
			switch permission {
			case "create":
				if wildcardPerm.Create {
					return true, nil
				}
			case "read":
				if wildcardPerm.Read {
					return true, nil
				}
			case "update":
				if wildcardPerm.Update {
					return true, nil
				}
			case "delete":
				if wildcardPerm.Delete {
					return true, nil
				}
			}
		}

		if modulePerm, exists := role.ModulePermissions[moduleName]; exists {

			switch permission {
			case "create":
				if modulePerm.Create {
					return true, nil
				}
			case "read":
				if modulePerm.Read {
					return true, nil
				}
			case "update":
				if modulePerm.Update {
					return true, nil
				}
			case "delete":
				if modulePerm.Delete {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func (s *RoleServiceImpl) GetFieldPermissions(ctx context.Context, userID primitive.ObjectID, moduleName string) (map[string]string, error) {
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
						// Union: read_write > read_only > none
						// Least Restrictive logic:
						// If any role says read_write, it's read_write.
						// If any role says read_only (and no read_write), it's read_only.
						if p == models.FieldPermReadWrite {
							finalPerms[field] = models.FieldPermReadWrite
						} else if p == models.FieldPermReadOnly {
							if current == models.FieldPermNone {
								finalPerms[field] = models.FieldPermReadOnly
							}
						}
					}
				}
			}
		}

		if role.FieldPermissions == nil || role.FieldPermissions[moduleName] == nil {
			// This role grants full access to this module's fields.
			// Return nil effectively.
			return nil, nil
		}
	}

	if !hasFieldRules {
		return nil, nil
	}

	return finalPerms, nil
}
