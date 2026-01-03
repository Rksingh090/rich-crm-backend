package settings

import (
	"context"
	"time"

	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/audit"
)

type SettingsService interface {
	GetEmailConfig(ctx context.Context) (*EmailConfig, error)
	UpdateEmailConfig(ctx context.Context, config EmailConfig) error
	GetGeneralConfig(ctx context.Context) (*GeneralConfig, error)
	UpdateGeneralConfig(ctx context.Context, config GeneralConfig) error
	GetFileSharingConfig(ctx context.Context) (*FileSharingConfig, error)
	UpdateFileSharingConfig(ctx context.Context, config FileSharingConfig) error
}

type SettingsServiceImpl struct {
	Repo         SettingsRepository
	AuditService audit.AuditService
}

func NewSettingsService(repo SettingsRepository, auditService audit.AuditService) SettingsService {
	return &SettingsServiceImpl{
		Repo:         repo,
		AuditService: auditService,
	}
}

func (s *SettingsServiceImpl) GetEmailConfig(ctx context.Context) (*EmailConfig, error) {
	settings, err := s.Repo.GetByType(ctx, SettingsTypeEmail)
	if err != nil {
		return nil, err
	}
	if settings == nil || settings.Email == nil {
		return nil, nil
	}
	return settings.Email, nil
}

func (s *SettingsServiceImpl) UpdateEmailConfig(ctx context.Context, config EmailConfig) error {
	oldConfig, _ := s.GetEmailConfig(ctx)

	settings := &Settings{
		Type:      SettingsTypeEmail,
		Email:     &config,
		UpdatedAt: time.Now(),
	}
	err := s.Repo.Upsert(ctx, settings)
	if err == nil {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionSettings, "settings", "email_config", map[string]common_models.Change{
			"email_config": {
				Old: oldConfig,
				New: config,
			},
		})
	}
	return err
}

func (s *SettingsServiceImpl) GetGeneralConfig(ctx context.Context) (*GeneralConfig, error) {
	settings, err := s.Repo.GetByType(ctx, SettingsTypeGeneral)
	if err != nil {
		return nil, err
	}
	if settings == nil || settings.General == nil {
		return &GeneralConfig{
			AppName: "Go CRM",
		}, nil
	}
	return settings.General, nil
}

func (s *SettingsServiceImpl) UpdateGeneralConfig(ctx context.Context, config GeneralConfig) error {
	oldConfig, _ := s.GetGeneralConfig(ctx)

	settings := &Settings{
		Type:      SettingsTypeGeneral,
		General:   &config,
		UpdatedAt: time.Now(),
	}
	err := s.Repo.Upsert(ctx, settings)
	if err == nil {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionSettings, "settings", "general_config", map[string]common_models.Change{
			"general_config": {
				Old: oldConfig,
				New: config,
			},
		})
	}
	return err
}

func (s *SettingsServiceImpl) GetFileSharingConfig(ctx context.Context) (*FileSharingConfig, error) {
	settings, err := s.Repo.GetByType(ctx, SettingsTypeFileSharing)
	if err != nil {
		return nil, err
	}
	if settings == nil || settings.FileSharing == nil {
		return &FileSharingConfig{
			Enabled:              true,
			MaxFileSizeMB:        10,
			AllowedFileTypes:     []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".png", ".jpg", ".jpeg", ".gif", ".txt", ".csv"},
			MaxFilesPerRecord:    20,
			EnabledModules:       []string{},
			AllowSharedDocuments: true,
		}, nil
	}
	return settings.FileSharing, nil
}

func (s *SettingsServiceImpl) UpdateFileSharingConfig(ctx context.Context, config FileSharingConfig) error {
	oldConfig, _ := s.GetFileSharingConfig(ctx)

	settings := &Settings{
		Type:        SettingsTypeFileSharing,
		FileSharing: &config,
		UpdatedAt:   time.Now(),
	}
	err := s.Repo.Upsert(ctx, settings)
	if err == nil {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionSettings, "settings", "file_sharing_config", map[string]common_models.Change{
			"file_sharing_config": {
				Old: oldConfig,
				New: config,
			},
		})
	}
	return err
}
