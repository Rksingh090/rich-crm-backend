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
}

type RoleServiceImpl struct {
	RoleRepo     repository.RoleRepository
	AuditService AuditService
}

func NewRoleService(roleRepo repository.RoleRepository, auditService AuditService) RoleService {
	return &RoleServiceImpl{
		RoleRepo:     roleRepo,
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

func (s *RoleServiceImpl) CheckModulePermission(ctx context.Context, roleIDs []string, moduleName string, permission string) (bool, error) {
	for _, roleID := range roleIDs {
		role, err := s.RoleRepo.FindByID(ctx, roleID)
		if err != nil {
			continue
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
