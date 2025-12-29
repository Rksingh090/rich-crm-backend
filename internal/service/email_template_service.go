package service

import (
	"context"
	"errors"
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"strings"
)

type EmailTemplateService interface {
	CreateTemplate(ctx context.Context, template *models.EmailTemplate) error
	GetTemplate(ctx context.Context, id string) (*models.EmailTemplate, error)
	ListTemplates(ctx context.Context, moduleName string, includeGlobal bool) ([]models.EmailTemplate, error)
	UpdateTemplate(ctx context.Context, template *models.EmailTemplate) error
	DeleteTemplate(ctx context.Context, id string) error
	GetModuleFields(ctx context.Context, moduleName string) ([]models.ModuleField, error)
	RenderTemplate(ctx context.Context, templateID string, record map[string]interface{}) (string, string, error)
}

type EmailTemplateServiceImpl struct {
	Repo         repository.EmailTemplateRepository
	ModuleRepo   repository.ModuleRepository
	AuditService AuditService
}

func NewEmailTemplateService(repo repository.EmailTemplateRepository, moduleRepo repository.ModuleRepository, auditService AuditService) EmailTemplateService {
	return &EmailTemplateServiceImpl{
		Repo:         repo,
		ModuleRepo:   moduleRepo,
		AuditService: auditService,
	}
}

func (s *EmailTemplateServiceImpl) CreateTemplate(ctx context.Context, template *models.EmailTemplate) error {
	if template.Name == "" {
		return errors.New("template name is required")
	}
	if template.Subject == "" {
		return errors.New("subject is required")
	}

	// Validate module if provided
	if template.ModuleName != "" {
		_, err := s.ModuleRepo.FindByName(ctx, template.ModuleName)
		if err != nil {
			return errors.New("invalid module name specified")
		}
	}

	err := s.Repo.Create(ctx, template)
	if err == nil {
		s.AuditService.LogChange(ctx, models.AuditActionTemplate, "email_templates", template.ID.Hex(), map[string]models.Change{
			"template": {
				New: template,
			},
		})
	}
	return err
}

func (s *EmailTemplateServiceImpl) GetTemplate(ctx context.Context, id string) (*models.EmailTemplate, error) {
	return s.Repo.GetByID(ctx, id)
}

func (s *EmailTemplateServiceImpl) ListTemplates(ctx context.Context, moduleName string, includeGlobal bool) ([]models.EmailTemplate, error) {
	// If includeGlobal is true, or if we are just listing specifically for a module/global context
	// The repo List method we implemented handles "moduleName or global" if moduleName is passed.
	// If moduleName is empty, it returns all? Wait, let's verify repo logic.
	// Repo logic: if moduleName != "", returns (module_name == moduleName OR module_name == "").
	// If moduleName == "", returns ALL (because filter is empty).

	// If we want ONLY global templates, we might need adjustments, or just filter in memory or pass a specific flag.
	// For "Settings > Email Templates", we likely want to see ALL templates, so we pass moduleName="".
	// For "Automation Rule > Select Template", we likely pass moduleName="Leads" and want Lead templates + Global templates.

	return s.Repo.List(ctx, moduleName)
}

func (s *EmailTemplateServiceImpl) UpdateTemplate(ctx context.Context, template *models.EmailTemplate) error {
	// Get old template for audit
	oldTemplate, _ := s.GetTemplate(ctx, template.ID.Hex())

	err := s.Repo.Update(ctx, template)
	if err == nil {
		s.AuditService.LogChange(ctx, models.AuditActionTemplate, "email_templates", template.ID.Hex(), map[string]models.Change{
			"template": {
				Old: oldTemplate,
				New: template,
			},
		})
	}
	return err
}

func (s *EmailTemplateServiceImpl) DeleteTemplate(ctx context.Context, id string) error {
	// Get old template for audit
	oldTemplate, _ := s.GetTemplate(ctx, id)

	err := s.Repo.Delete(ctx, id)
	if err == nil {
		name := id
		if oldTemplate != nil {
			name = oldTemplate.Name
		}
		s.AuditService.LogChange(ctx, models.AuditActionTemplate, "email_templates", name, map[string]models.Change{
			"template": {
				Old: oldTemplate,
				New: "DELETED",
			},
		})
	}
	return err
}

func (s *EmailTemplateServiceImpl) GetModuleFields(ctx context.Context, moduleName string) ([]models.ModuleField, error) {
	module, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, errors.New("module not found")
	}
	return module.Fields, nil
}

func (s *EmailTemplateServiceImpl) RenderTemplate(ctx context.Context, templateID string, record map[string]interface{}) (string, string, error) {
	template, err := s.Repo.GetByID(ctx, templateID)
	if err != nil {
		return "", "", err
	}

	subject := s.replacePlaceholders(template.Subject, record)
	body := s.replacePlaceholders(template.Body, record)

	return subject, body, nil
}

func (s *EmailTemplateServiceImpl) replacePlaceholders(text string, record map[string]interface{}) string {
	for key, value := range record {
		placeholder := fmt.Sprintf("{{%s}}", key)
		replacement := fmt.Sprintf("%v", value)
		text = strings.ReplaceAll(text, placeholder, replacement)
	}
	return text
}
