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
