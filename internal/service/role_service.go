package service

import (
	"context"
	"time"

	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RoleServiceImpl struct {
	RoleRepo repository.RoleRepository
}

func NewRoleServiceImpl(roleRepo repository.RoleRepository) *RoleServiceImpl {
	return &RoleServiceImpl{
		RoleRepo: roleRepo,
	}
}

func (s *RoleServiceImpl) CreateRole(ctx context.Context, name string, permissions []string) (*models.Role, error) {
	role := models.Role{
		ID:          primitive.NewObjectID(),
		Name:        name,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.RoleRepo.Create(ctx, &role); err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *RoleServiceImpl) GetRoleByName(ctx context.Context, name string) (*models.Role, error) {
	return s.RoleRepo.FindByName(ctx, name)
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
