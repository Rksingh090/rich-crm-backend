package app

import (
	"context"
	"go-crm/internal/common/models"
)

type AppService interface {
	SyncApps(ctx context.Context, apps []models.Application) error
	ListApps(ctx context.Context) ([]models.Application, error)
}

type appServiceImpl struct {
	repo AppRepository
}

func NewAppService(repo AppRepository) AppService {
	return &appServiceImpl{repo: repo}
}

func (s *appServiceImpl) SyncApps(ctx context.Context, apps []models.Application) error {
	for _, app := range apps {
		existing, err := s.repo.GetByName(ctx, app.Name)
		if err == nil {
			app.ID = existing.ID
			if err := s.repo.Update(ctx, &app); err != nil {
				return err
			}
		} else {
			if err := s.repo.Create(ctx, &app); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *appServiceImpl) ListApps(ctx context.Context) ([]models.Application, error) {
	return s.repo.List(ctx)
}
