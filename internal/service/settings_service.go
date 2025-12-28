package service

import (
	"context"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"time"
)

type SettingsService interface {
	GetEmailConfig(ctx context.Context) (*models.EmailConfig, error)
	UpdateEmailConfig(ctx context.Context, config models.EmailConfig) error
	GetGeneralConfig(ctx context.Context) (*models.GeneralConfig, error)
	UpdateGeneralConfig(ctx context.Context, config models.GeneralConfig) error
}

type SettingsServiceImpl struct {
	Repo repository.SettingsRepository
}

func NewSettingsService(repo repository.SettingsRepository) SettingsService {
	return &SettingsServiceImpl{
		Repo: repo,
	}
}

func (s *SettingsServiceImpl) GetEmailConfig(ctx context.Context) (*models.EmailConfig, error) {
	settings, err := s.Repo.GetByType(ctx, models.SettingsTypeEmail)
	if err != nil {
		return nil, err
	}
	if settings == nil || settings.Email == nil {
		return nil, nil // Not Configured
	}
	return settings.Email, nil
}

func (s *SettingsServiceImpl) UpdateEmailConfig(ctx context.Context, config models.EmailConfig) error {
	settings := &models.Settings{
		Type:      models.SettingsTypeEmail,
		Email:     &config,
		UpdatedAt: time.Now(),
	}
	return s.Repo.Upsert(ctx, settings)
}

func (s *SettingsServiceImpl) GetGeneralConfig(ctx context.Context) (*models.GeneralConfig, error) {
	settings, err := s.Repo.GetByType(ctx, models.SettingsTypeGeneral)
	if err != nil {
		return nil, err
	}
	if settings == nil || settings.General == nil {
		// Return default empty config if not found
		return &models.GeneralConfig{
			AppName: "Go CRM",
		}, nil
	}
	return settings.General, nil
}

func (s *SettingsServiceImpl) UpdateGeneralConfig(ctx context.Context, config models.GeneralConfig) error {
	settings := &models.Settings{
		Type:      models.SettingsTypeGeneral,
		General:   &config,
		UpdatedAt: time.Now(),
	}
	return s.Repo.Upsert(ctx, settings)
}
