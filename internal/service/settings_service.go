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
	GetFileSharingConfig(ctx context.Context) (*models.FileSharingConfig, error)
	UpdateFileSharingConfig(ctx context.Context, config models.FileSharingConfig) error
}

type SettingsServiceImpl struct {
	Repo         repository.SettingsRepository
	AuditService AuditService
}

func NewSettingsService(repo repository.SettingsRepository, auditService AuditService) SettingsService {
	return &SettingsServiceImpl{
		Repo:         repo,
		AuditService: auditService,
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
	// Fetch old config for audit log
	oldConfig, _ := s.GetEmailConfig(ctx)

	settings := &models.Settings{
		Type:      models.SettingsTypeEmail,
		Email:     &config,
		UpdatedAt: time.Now(),
	}
	err := s.Repo.Upsert(ctx, settings)
	if err == nil {
		s.AuditService.LogChange(ctx, models.AuditActionSettings, "settings", "email_config", map[string]models.Change{
			"email_config": {
				Old: oldConfig,
				New: config,
			},
		})
	}
	return err
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
	// Fetch old config for audit log
	oldConfig, _ := s.GetGeneralConfig(ctx)

	settings := &models.Settings{
		Type:      models.SettingsTypeGeneral,
		General:   &config,
		UpdatedAt: time.Now(),
	}
	err := s.Repo.Upsert(ctx, settings)
	if err == nil {
		s.AuditService.LogChange(ctx, models.AuditActionSettings, "settings", "general_config", map[string]models.Change{
			"general_config": {
				Old: oldConfig,
				New: config,
			},
		})
	}
	return err
}

func (s *SettingsServiceImpl) GetFileSharingConfig(ctx context.Context) (*models.FileSharingConfig, error) {
	settings, err := s.Repo.GetByType(ctx, models.SettingsTypeFileSharing)
	if err != nil {
		return nil, err
	}
	if settings == nil || settings.FileSharing == nil {
		// Return default config if not found
		return &models.FileSharingConfig{
			Enabled:              true,
			MaxFileSizeMB:        10,
			AllowedFileTypes:     []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".png", ".jpg", ".jpeg", ".gif", ".txt", ".csv"},
			MaxFilesPerRecord:    20,
			EnabledModules:       []string{}, // Empty = all modules
			AllowSharedDocuments: true,
		}, nil
	}
	return settings.FileSharing, nil
}

func (s *SettingsServiceImpl) UpdateFileSharingConfig(ctx context.Context, config models.FileSharingConfig) error {
	// Fetch old config for audit log
	oldConfig, _ := s.GetFileSharingConfig(ctx)

	settings := &models.Settings{
		Type:        models.SettingsTypeFileSharing,
		FileSharing: &config,
		UpdatedAt:   time.Now(),
	}
	err := s.Repo.Upsert(ctx, settings)
	if err == nil {
		s.AuditService.LogChange(ctx, models.AuditActionSettings, "settings", "file_sharing_config", map[string]models.Change{
			"file_sharing_config": {
				Old: oldConfig,
				New: config,
			},
		})
	}
	return err
}
