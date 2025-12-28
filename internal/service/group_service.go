package service

import (
	"context"
	"errors"
	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GroupService struct {
	repo *repository.GroupRepository
}

func NewGroupService(repo *repository.GroupRepository) *GroupService {
	return &GroupService{repo: repo}
}

func (s *GroupService) CreateGroup(ctx context.Context, group *models.Group) error {
	if group.Name == "" {
		return errors.New("group name is required")
	}
	return s.repo.Create(ctx, group)
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

	return s.repo.Update(ctx, id, group)
}

func (s *GroupService) DeleteGroup(ctx context.Context, id primitive.ObjectID) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.IsSystem {
		return errors.New("cannot delete system group")
	}

	return s.repo.Delete(ctx, id)
}

func (s *GroupService) AddMember(ctx context.Context, groupID, userID primitive.ObjectID) error {
	return s.repo.AddMember(ctx, groupID, userID)
}

func (s *GroupService) RemoveMember(ctx context.Context, groupID, userID primitive.ObjectID) error {
	return s.repo.RemoveMember(ctx, groupID, userID)
}

func (s *GroupService) GetUserGroups(ctx context.Context, userID primitive.ObjectID) ([]models.Group, error) {
	return s.repo.FindByMember(ctx, userID)
}
