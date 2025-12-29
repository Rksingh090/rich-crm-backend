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
	Repo           repository.AutomationRepository
	ActionExecutor ActionExecutor
	AuditService   AuditService
}

func NewAutomationService(repo repository.AutomationRepository, actionExecutor ActionExecutor, auditService AuditService) AutomationService {
	return &AutomationServiceImpl{
		Repo:           repo,
		ActionExecutor: actionExecutor,
		AuditService:   auditService,
	}
}

func (s *AutomationServiceImpl) CreateRule(ctx context.Context, rule *models.AutomationRule) error {
	err := s.Repo.Create(ctx, rule)
	if err == nil {
		s.AuditService.LogChange(ctx, models.AuditActionAutomation, "automation", rule.ID.Hex(), map[string]models.Change{
			"rule": {New: rule},
		})
	}
	return err
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
	// Get old rule for audit
	oldRule, _ := s.GetRule(ctx, rule.ID.Hex())

	err := s.Repo.Update(ctx, rule)
	if err == nil {
		s.AuditService.LogChange(ctx, models.AuditActionAutomation, "automation", rule.ID.Hex(), map[string]models.Change{
			"rule": {Old: oldRule, New: rule},
		})
	}
	return err
}

func (s *AutomationServiceImpl) DeleteRule(ctx context.Context, id string) error {
	// Get old rule for audit
	oldRule, _ := s.GetRule(ctx, id)

	err := s.Repo.Delete(ctx, id)
	if err == nil {
		name := id
		if oldRule != nil {
			name = oldRule.Name
		}
		s.AuditService.LogChange(ctx, models.AuditActionAutomation, "automation", name, map[string]models.Change{
			"rule": {Old: oldRule, New: "DELETED"},
		})
	}
	return err
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
	// Delegate to centralized ActionExecutor
	return s.ActionExecutor.ExecuteActions(ctx, actions, moduleName, record)
}

// executeDynamicScript is kept for backward compatibility but should be moved to ActionExecutor
// This is now deprecated and will be removed in future versions
