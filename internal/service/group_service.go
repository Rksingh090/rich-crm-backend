package service

import (
	"context"
	"errors"
	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GroupService struct {
	repo         *repository.GroupRepository
	auditService AuditService
}

func NewGroupService(repo *repository.GroupRepository, auditService AuditService) *GroupService {
	return &GroupService{
		repo:         repo,
		auditService: auditService,
	}
}

func (s *GroupService) CreateGroup(ctx context.Context, group *models.Group) error {
	if group.Name == "" {
		return errors.New("group name is required")
	}
	err := s.repo.Create(ctx, group)
	if err == nil {
		s.auditService.LogChange(ctx, models.AuditActionGroup, "groups", group.ID.Hex(), map[string]models.Change{
			"group": {New: group},
		})
	}
	return err
}

func (s *GroupService) GetAllGroups(ctx context.Context) ([]models.Group, error) {
	return s.repo.FindAll(ctx)
}

func (s *GroupService) GetGroupByID(ctx context.Context, id primitive.ObjectID) (*models.Group, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *GroupService) UpdateGroup(ctx context.Context, id primitive.ObjectID, group *models.Group) error {
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
		s.auditService.LogChange(ctx, models.AuditActionGroup, "groups", id.Hex(), map[string]models.Change{
			"group": {Old: existing, New: group},
		})
	}
	return err
}

func (s *GroupService) DeleteGroup(ctx context.Context, id primitive.ObjectID) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.IsSystem {
		return errors.New("cannot delete system group")
	}

	err = s.repo.Delete(ctx, id)
	if err == nil {
		s.auditService.LogChange(ctx, models.AuditActionGroup, "groups", id.Hex(), map[string]models.Change{
			"group": {Old: existing, New: "DELETED"},
		})
	}
	return err
}

func (s *GroupService) AddMember(ctx context.Context, groupID, userID primitive.ObjectID) error {
	err := s.repo.AddMember(ctx, groupID, userID)
	if err == nil {
		s.auditService.LogChange(ctx, models.AuditActionGroup, "groups", groupID.Hex(), map[string]models.Change{
			"member_added": {New: userID.Hex()},
		})
	}
	return err
}

func (s *GroupService) RemoveMember(ctx context.Context, groupID, userID primitive.ObjectID) error {
	err := s.repo.RemoveMember(ctx, groupID, userID)
	if err == nil {
		s.auditService.LogChange(ctx, models.AuditActionGroup, "groups", groupID.Hex(), map[string]models.Change{
			"member_removed": {Old: userID.Hex()},
		})
	}
	return err
}

func (s *GroupService) GetUserGroups(ctx context.Context, userID primitive.ObjectID) ([]models.Group, error) {
	return s.repo.FindByMember(ctx, userID)
}
