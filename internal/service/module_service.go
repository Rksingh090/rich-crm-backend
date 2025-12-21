package service

import (
	"context"
	"errors"
	"time"

	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ModuleServiceImpl struct {
	Repo repository.ModuleRepository
}

func NewModuleServiceImpl(repo repository.ModuleRepository) *ModuleServiceImpl {
	return &ModuleServiceImpl{
		Repo: repo,
	}
}

func (s *ModuleServiceImpl) CreateModule(ctx context.Context, module *models.Module) error {
	// Basic Validation
	if module.Name == "" || module.Label == "" {
		return errors.New("module name and label are required")
	}

	// Check if already exists
	if _, err := s.Repo.FindByName(ctx, module.Name); err == nil {
		return errors.New("module with this name already exists")
	}

	module.ID = primitive.NewObjectID()
	module.CreatedAt = time.Now()
	module.UpdatedAt = time.Now()

	return s.Repo.Create(ctx, module)
}

func (s *ModuleServiceImpl) GetModuleByName(ctx context.Context, name string) (*models.Module, error) {
	return s.Repo.FindByName(ctx, name)
}

func (s *ModuleServiceImpl) ListModules(ctx context.Context) ([]models.Module, error) {
	return s.Repo.List(ctx)
}

func (s *ModuleServiceImpl) UpdateModule(ctx context.Context, module *models.Module) error {
	module.UpdatedAt = time.Now()
	// In real app, we might check if module exists first or validate schema changes
	return s.Repo.Update(ctx, module)
}

func (s *ModuleServiceImpl) DeleteModule(ctx context.Context, name string) error {
	return s.Repo.Delete(ctx, name)
}
