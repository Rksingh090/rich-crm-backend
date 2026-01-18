package automation

import (
	"context"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/core/action"
	"go-crm/internal/core/audit"
	"strings"
)

type AutomationService interface {
	CreateRule(ctx context.Context, rule *AutomationRule) error
	GetRule(ctx context.Context, id string) (*AutomationRule, error)
	ListRules(ctx context.Context, moduleID string) ([]AutomationRule, error)
	UpdateRule(ctx context.Context, rule *AutomationRule) error
	DeleteRule(ctx context.Context, id string) error

	// Core Logic
	ExecuteFromTrigger(ctx context.Context, moduleName string, record map[string]any, triggerType string) error
}

type AutomationServiceImpl struct {
	Repo           AutomationRepository
	ActionExecutor action.ActionExecutor
	AuditService   audit.AuditService
}

func NewAutomationService(repo AutomationRepository, actionExecutor action.ActionExecutor, auditService audit.AuditService) *AutomationServiceImpl {

	return &AutomationServiceImpl{
		Repo:           repo,
		ActionExecutor: actionExecutor,
		AuditService:   auditService,
	}
}

func (s *AutomationServiceImpl) CreateRule(ctx context.Context, rule *AutomationRule) error {
	err := s.Repo.Create(ctx, rule)
	if err == nil {
		s.AuditService.LogChange(ctx, common_models.AuditActionAutomation, "automation", rule.ID.Hex(), map[string]common_models.Change{
			"rule": {New: rule},
		})
	}
	return err
}

func (s *AutomationServiceImpl) GetRule(ctx context.Context, id string) (*AutomationRule, error) {
	return s.Repo.GetByID(ctx, id)
}

func (s *AutomationServiceImpl) ListRules(ctx context.Context, moduleID string) ([]AutomationRule, error) {
	if moduleID != "" {
		return s.Repo.GetByModule(ctx, moduleID)
	}
	return s.Repo.List(ctx)
}

func (s *AutomationServiceImpl) UpdateRule(ctx context.Context, rule *AutomationRule) error {
	// Get old rule for audit
	oldRule, _ := s.GetRule(ctx, rule.ID.Hex())

	err := s.Repo.Update(ctx, rule)
	if err == nil {
		s.AuditService.LogChange(ctx, common_models.AuditActionAutomation, "automation", rule.ID.Hex(), map[string]common_models.Change{
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
		s.AuditService.LogChange(ctx, common_models.AuditActionAutomation, "automation", name, map[string]common_models.Change{
			"rule": {Old: oldRule, New: "DELETED"},
		})
	}
	return err
}

func (s *AutomationServiceImpl) ExecuteFromTrigger(ctx context.Context, moduleName string, record map[string]any, triggerType string) error {
	rules, err := s.Repo.GetByModule(ctx, moduleName)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if !rule.Active {
			continue
		}

		// Check if trigger matches
		// 1. Exact match (e.g. "create" == "create")
		// 2. "create_or_update" matches if incoming trigger is "create" or "update"
		isMatch := rule.TriggerType == triggerType
		if rule.TriggerType == "create_or_update" && (triggerType == "create" || triggerType == "update") {
			isMatch = true
		}

		if !isMatch {
			continue
		}

		if s.evaluateConditions(rule.Conditions, record) {
			// Inject TenantID into context for action execution
			// This is required for fetching settings and other tenant-specific operations
			execCtx := context.WithValue(ctx, common_models.TenantIDKey, rule.TenantID.Hex())
			if err := s.executeActions(execCtx, rule.Actions, moduleName, record); err != nil {
				fmt.Printf("Error executing automation rule '%s': %v\n", rule.Name, err)
			}
		}
	}
	return nil
}

func (s *AutomationServiceImpl) evaluateConditions(conditions []RuleCondition, record map[string]any) bool {
	for _, cond := range conditions {
		val, exists := record[cond.Field]
		if !exists {
			return false
		}

		match := false
		switch cond.Operator {
		case OperatorEquals:
			match = fmt.Sprintf("%v", val) == fmt.Sprintf("%v", cond.Value)
		case OperatorNotEquals:
			match = fmt.Sprintf("%v", val) != fmt.Sprintf("%v", cond.Value)
		case OperatorContains:
			match = strings.Contains(fmt.Sprintf("%v", val), fmt.Sprintf("%v", cond.Value))
		default:
			match = false
		}

		if !match {
			return false
		}
	}
	return true
}

func (s *AutomationServiceImpl) executeActions(ctx context.Context, actions []action.Action, moduleName string, record map[string]any) error {
	return s.ActionExecutor.ExecuteActions(ctx, actions, moduleName, record)
}
