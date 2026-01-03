package extension

import (
	"context"
	"time"
)

type ExtensionService interface {
	InstallExtension(ctx context.Context, id string) error
	UninstallExtension(ctx context.Context, id string) error
	ListExtensions(ctx context.Context, onlyInstalled bool) ([]Extension, error)
	GetExtension(ctx context.Context, id string) (*Extension, error)
	CreateExtension(ctx context.Context, ext *Extension) error
}

type ExtensionServiceImpl struct {
	Repo ExtensionRepository
}

func NewExtensionService(repo ExtensionRepository) ExtensionService {
	return &ExtensionServiceImpl{
		Repo: repo,
	}
}

func (s *ExtensionServiceImpl) InstallExtension(ctx context.Context, id string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"installed":    true,
		"installed_at": &now,
		"status":       ExtensionStatusActive,
	}
	return s.Repo.Update(ctx, id, updates)
}

func (s *ExtensionServiceImpl) UninstallExtension(ctx context.Context, id string) error {
	updates := map[string]interface{}{
		"installed":    false,
		"installed_at": nil,
		"status":       ExtensionStatusInactive,
	}
	return s.Repo.Update(ctx, id, updates)
}

func (s *ExtensionServiceImpl) ListExtensions(ctx context.Context, onlyInstalled bool) ([]Extension, error) {
	return s.Repo.List(ctx, onlyInstalled)
}

func (s *ExtensionServiceImpl) GetExtension(ctx context.Context, id string) (*Extension, error) {
	return s.Repo.GetByID(ctx, id)
}

func (s *ExtensionServiceImpl) CreateExtension(ctx context.Context, ext *Extension) error {
	return s.Repo.Create(ctx, ext)
}
