package email_template

import (
	"context"
	"errors"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/email"
	"go-crm/internal/features/module"
	"strings"
)

type EmailTemplateService interface {
	CreateTemplate(ctx context.Context, template *EmailTemplate) error
	GetTemplate(ctx context.Context, id string) (*EmailTemplate, error)
	ListTemplates(ctx context.Context, moduleName string, includeGlobal bool) ([]EmailTemplate, error)
	UpdateTemplate(ctx context.Context, template *EmailTemplate) error
	DeleteTemplate(ctx context.Context, id string) error
	GetModuleFields(ctx context.Context, moduleName string) ([]module.ModuleField, error)
	RenderTemplate(ctx context.Context, templateID string, record map[string]interface{}) (string, string, error)
	SendTestEmail(ctx context.Context, templateID string, to string, testData map[string]interface{}) error
}

type EmailTemplateServiceImpl struct {
	Repo         EmailTemplateRepository
	ModuleRepo   module.ModuleRepository
	AuditService audit.AuditService
	EmailService email.EmailService
}

func NewEmailTemplateService(
	repo EmailTemplateRepository,
	moduleRepo module.ModuleRepository,
	auditService audit.AuditService,
	emailService email.EmailService,
) EmailTemplateService {
	return &EmailTemplateServiceImpl{
		Repo:         repo,
		ModuleRepo:   moduleRepo,
		AuditService: auditService,
		EmailService: emailService,
	}
}

func (s *EmailTemplateServiceImpl) CreateTemplate(ctx context.Context, template *EmailTemplate) error {
	if template.Name == "" {
		return errors.New("template name is required")
	}
	if template.Subject == "" {
		return errors.New("subject is required")
	}

	if template.ModuleName != "" {
		_, err := s.ModuleRepo.FindByName(ctx, template.ModuleName)
		if err != nil {
			return errors.New("invalid module name specified")
		}
	}

	err := s.Repo.Create(ctx, template)
	if err == nil {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionTemplate, "email_templates", template.ID.Hex(), map[string]common_models.Change{
			"template": {
				New: template,
			},
		})
	}
	return err
}

func (s *EmailTemplateServiceImpl) GetTemplate(ctx context.Context, id string) (*EmailTemplate, error) {
	return s.Repo.GetByID(ctx, id)
}

func (s *EmailTemplateServiceImpl) ListTemplates(ctx context.Context, moduleName string, includeGlobal bool) ([]EmailTemplate, error) {
	return s.Repo.List(ctx, moduleName)
}

func (s *EmailTemplateServiceImpl) UpdateTemplate(ctx context.Context, template *EmailTemplate) error {
	oldTemplate, _ := s.GetTemplate(ctx, template.ID.Hex())

	err := s.Repo.Update(ctx, template)
	if err == nil {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionTemplate, "email_templates", template.ID.Hex(), map[string]common_models.Change{
			"template": {
				Old: oldTemplate,
				New: template,
			},
		})
	}
	return err
}

func (s *EmailTemplateServiceImpl) DeleteTemplate(ctx context.Context, id string) error {
	oldTemplate, _ := s.GetTemplate(ctx, id)

	err := s.Repo.Delete(ctx, id)
	if err == nil {
		name := id
		if oldTemplate != nil {
			name = oldTemplate.Name
		}
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionTemplate, "email_templates", name, map[string]common_models.Change{
			"template": {
				Old: oldTemplate,
				New: "DELETED",
			},
		})
	}
	return err
}

func (s *EmailTemplateServiceImpl) GetModuleFields(ctx context.Context, moduleName string) ([]module.ModuleField, error) {
	mod, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return nil, err
	}
	if mod == nil {
		return nil, errors.New("module not found")
	}
	return mod.Fields, nil
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

func (s *EmailTemplateServiceImpl) SendTestEmail(ctx context.Context, templateID string, to string, testData map[string]interface{}) error {
	subject, body, err := s.RenderTemplate(ctx, templateID, testData)
	if err != nil {
		return err
	}

	return s.EmailService.SendEmail(ctx, []string{to}, subject, body)
}
