package service

import (
	"context"
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"strings"
)

type AutomationService interface {
	CreateRule(ctx context.Context, rule *models.AutomationRule) error
	GetRule(ctx context.Context, id string) (*models.AutomationRule, error)
	ListRules(ctx context.Context, moduleID string) ([]models.AutomationRule, error)
	UpdateRule(ctx context.Context, rule *models.AutomationRule) error
	DeleteRule(ctx context.Context, id string) error

	// Core Logic
	ExecuteFromTrigger(ctx context.Context, moduleName string, record map[string]interface{}, triggerType string) error
}

type AutomationServiceImpl struct {
	Repo         repository.AutomationRepository
	RecordRepo   repository.RecordRepository
	AuditService AuditService
	EmailService EmailService
}

func NewAutomationService(repo repository.AutomationRepository, recordRepo repository.RecordRepository, auditService AuditService, emailService EmailService) AutomationService {
	return &AutomationServiceImpl{
		Repo:         repo,
		RecordRepo:   recordRepo,
		AuditService: auditService,
		EmailService: emailService,
	}
}

func (s *AutomationServiceImpl) CreateRule(ctx context.Context, rule *models.AutomationRule) error {
	return s.Repo.Create(ctx, rule)
}

func (s *AutomationServiceImpl) GetRule(ctx context.Context, id string) (*models.AutomationRule, error) {
	return s.Repo.GetByID(ctx, id)
}

func (s *AutomationServiceImpl) ListRules(ctx context.Context, moduleID string) ([]models.AutomationRule, error) {
	if moduleID != "" {
		return s.Repo.GetByModule(ctx, moduleID)
	}
	return s.Repo.List(ctx)
}

func (s *AutomationServiceImpl) UpdateRule(ctx context.Context, rule *models.AutomationRule) error {
	return s.Repo.Update(ctx, rule)
}

func (s *AutomationServiceImpl) DeleteRule(ctx context.Context, id string) error {
	return s.Repo.Delete(ctx, id)
}

func (s *AutomationServiceImpl) ExecuteFromTrigger(ctx context.Context, moduleName string, record map[string]interface{}, triggerType string) error {
	// 1. Fetch Active Rules for Module & Trigger
	// We can filter in memory or add specific query to repo. For now, filter in memory.
	rules, err := s.Repo.GetByModule(ctx, moduleName)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if !rule.Active || rule.TriggerType != triggerType {
			continue
		}

		// 2. Evaluate Conditions
		if s.evaluateConditions(rule.Conditions, record) {
			// 3. Execute Actions
			if err := s.executeActions(ctx, rule.Actions, moduleName, record); err != nil {
				fmt.Printf("Error executing automation rule '%s': %v\n", rule.Name, err)
				// Continue to next rule even if one fails
			}
		}
	}
	return nil
}

func (s *AutomationServiceImpl) evaluateConditions(conditions []models.RuleCondition, record map[string]interface{}) bool {
	for _, cond := range conditions {
		val, exists := record[cond.Field]
		if !exists {
			// If field doesn't exist, condition fails unless checking for "not exist"?
			// For now assume fail
			return false
		}

		// Basic comparison (string/number/bool) logic
		// This needs to be robust for types (float64 vs int vs string)
		match := false
		switch cond.Operator {
		case models.OperatorEquals:
			match = fmt.Sprintf("%v", val) == fmt.Sprintf("%v", cond.Value)
		case models.OperatorNotEquals:
			match = fmt.Sprintf("%v", val) != fmt.Sprintf("%v", cond.Value)
		case models.OperatorContains:
			match = strings.Contains(fmt.Sprintf("%v", val), fmt.Sprintf("%v", cond.Value))
		// Add primitive GT/LT logic if needed (requires type assertions)
		default:
			match = false
		}

		if !match {
			return false
		}
	}
	return true
}

func (s *AutomationServiceImpl) executeActions(ctx context.Context, actions []models.RuleAction, moduleName string, record map[string]interface{}) error {
	for _, action := range actions {
		switch action.Type {
		case models.ActionCreateTask:
			// Config: { "subject": "Task Name", "due_in_hours": 24 }
			// Logic: Create record in "tasks" module
			// This requires "tasks" module to exist and "RecordRepo" usage
			// For MVP: Just log or implement generic record creation if easy
			fmt.Printf("Automation Action: Create Task %v\n", action.Config)

		case models.ActionUpdateField:
			// Config: { "field": "status", "value": "Contacted" }
			fieldName, _ := action.Config["field"].(string)
			newVal := action.Config["value"]
			if fieldName != "" {
				if idRaw, ok := record["_id"]; ok {
					// Need to convert objectID to string hex
					idHex := fmt.Sprintf("%v", idRaw)
					// Use RecordRepo directly? Or Service?
					// RecordService calls AutomationService, so calling RecordService here causes Cycle.
					// Must use RecordRepo.

					// Dangerous: We are bypassing Service validation (schema, readonly, etc.)
					// But usually automation has "sudo" rights.
					err := s.RecordRepo.Update(ctx, moduleName, strings.TrimPrefix(strings.TrimSuffix(idHex, ")"), "ObjectID(\""), map[string]interface{}{fieldName: newVal})
					if err != nil {
						return err
					}
					// fmt.Printf("Automation Action: Updated field %s to %v\n", fieldName, newVal)

					// Log to Audit
					if s.AuditService != nil {
						changes := map[string]models.Change{
							fieldName: {New: newVal},
						}
						// RecordID? We have it in `record["_id"]`, but need to parse hex
						recID := strings.TrimPrefix(strings.TrimSuffix(idHex, ")"), "ObjectID(\"")
						_ = s.AuditService.LogChange(ctx, models.AuditActionAutomation, moduleName, recID, changes)
					}
				}
			}

		case models.ActionSendEmail:
			// Config: { "to": "...", "subject": "...", "body": "..." }
			fmt.Printf("Automation Action: Send Email %v\n", action.Config)

			if s.EmailService != nil {
				// Parse recipients. Config 'to' might be string (comma user) or array
				toStr, ok := action.Config["to"].(string)
				if ok && toStr != "" {
					to := []string{toStr} // TODO: Handle multiple/comma separated
					subject, _ := action.Config["subject"].(string)
					body, _ := action.Config["body"].(string)

					// Send in background? Or sync?
					// Sync ensures we know if it failed, but might block.
					// Let's assume sync for now.
					if err := s.EmailService.SendEmail(ctx, to, subject, body); err != nil {
						fmt.Printf("Failed to send email: %v\n", err)
						// Don't return error to stop other actions? Or return?
						// Just log for now.
					}
				}
			}

			// Log to Audit
			if s.AuditService != nil {
				// Try to get Record ID
				recID := ""
				if idRaw, ok := record["_id"]; ok {
					idHex := fmt.Sprintf("%v", idRaw)
					recID = strings.TrimPrefix(strings.TrimSuffix(idHex, ")"), "ObjectID(\"")
				}
				_ = s.AuditService.LogChange(ctx, models.AuditActionAutomation, moduleName, recID, map[string]models.Change{
					"action": {New: fmt.Sprintf("Sent Email: %v", action.Config["subject"])},
				})
			}
		}
	}
	return nil
}
