package group

import (
	"context"
	"errors"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/audit"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GroupService interface {
	CreateGroup(ctx context.Context, group *Group) error
	GetAllGroups(ctx context.Context) ([]Group, error)
	GetGroupByID(ctx context.Context, id primitive.ObjectID) (*Group, error)
	UpdateGroup(ctx context.Context, id primitive.ObjectID, group *Group) error
	DeleteGroup(ctx context.Context, id primitive.ObjectID) error
	AddMember(ctx context.Context, groupID, userID primitive.ObjectID) error
	RemoveMember(ctx context.Context, groupID, userID primitive.ObjectID) error
	GetUserGroups(ctx context.Context, userID primitive.ObjectID) ([]Group, error)
}

type GroupServiceImpl struct {
	repo         GroupRepository
	auditService audit.AuditService
}

func NewGroupService(repo GroupRepository, auditService audit.AuditService) GroupService {
	return &GroupServiceImpl{
		repo:         repo,
		auditService: auditService,
	}
}

func (s *GroupServiceImpl) CreateGroup(ctx context.Context, group *Group) error {
	if group.Name == "" {
		return errors.New("group name is required")
	}
	err := s.repo.Create(ctx, group)
	if err == nil {
		_ = s.auditService.LogChange(ctx, common_models.AuditActionGroup, "groups", group.ID.Hex(), map[string]common_models.Change{
			"group": {New: group},
		})
	}
	return err
}

func (s *GroupServiceImpl) GetAllGroups(ctx context.Context) ([]Group, error) {
	return s.repo.FindAll(ctx)
}

func (s *GroupServiceImpl) GetGroupByID(ctx context.Context, id primitive.ObjectID) (*Group, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *GroupServiceImpl) UpdateGroup(ctx context.Context, id primitive.ObjectID, group *Group) error {
	if group.Name == "" {
		return errors.New("group name is required")
	}

	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.IsSystem {
		return errors.New("cannot modify system group")
	}

	err = s.repo.Update(ctx, id, group)
	if err == nil {
		_ = s.auditService.LogChange(ctx, common_models.AuditActionGroup, "groups", id.Hex(), map[string]common_models.Change{
			"group": {Old: existing, New: group},
		})
	}
	return err
}

func (s *GroupServiceImpl) DeleteGroup(ctx context.Context, id primitive.ObjectID) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.IsSystem {
		return errors.New("cannot delete system group")
	}

	err = s.repo.Delete(ctx, id)
	if err == nil {
		_ = s.auditService.LogChange(ctx, common_models.AuditActionGroup, "groups", id.Hex(), map[string]common_models.Change{
			"group": {Old: existing, New: "DELETED"},
		})
	}
	return err
}

func (s *GroupServiceImpl) AddMember(ctx context.Context, groupID, userID primitive.ObjectID) error {
	err := s.repo.AddMember(ctx, groupID, userID)
	if err == nil {
		_ = s.auditService.LogChange(ctx, common_models.AuditActionGroup, "groups", groupID.Hex(), map[string]common_models.Change{
			"member_added": {New: userID.Hex()},
		})
	}
	return err
}

func (s *GroupServiceImpl) RemoveMember(ctx context.Context, groupID, userID primitive.ObjectID) error {
	err := s.repo.RemoveMember(ctx, groupID, userID)
	if err == nil {
		_ = s.auditService.LogChange(ctx, common_models.AuditActionGroup, "groups", groupID.Hex(), map[string]common_models.Change{
			"member_removed": {Old: userID.Hex()},
		})
	}
	return err
}

func (s *GroupServiceImpl) GetUserGroups(ctx context.Context, userID primitive.ObjectID) ([]Group, error) {
	return s.repo.FindByMember(ctx, userID)
}
