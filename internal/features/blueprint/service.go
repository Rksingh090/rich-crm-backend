package blueprint

import (
	"context"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/automation"
	"go-crm/internal/features/record"
	"strings"
	"time"
)

type Service interface {
	Create(ctx context.Context, blueprint *Blueprint) error
	Update(ctx context.Context, id string, blueprint *Blueprint) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*Blueprint, error)
	List(ctx context.Context, filter BlueprintFilter) ([]Blueprint, error)
	GetActiveByModule(ctx context.Context, module string) (*Blueprint, error)

	// Core Logic
	ExecuteTransition(ctx context.Context, blueprintID string, transitionID string, recordID string, data map[string]interface{}) error
	ValidateTransition(ctx context.Context, blueprintID string, transitionID string, recordID string) (bool, error)
}

type ServiceImpl struct {
	repo       Repository
	recordRepo record.RecordRepository
	executor   automation.ActionExecutor
}

func NewService(repo Repository, recordRepo record.RecordRepository, executor automation.ActionExecutor) Service {
	return &ServiceImpl{
		repo:       repo,
		recordRepo: recordRepo,
		executor:   executor,
	}
}

func (s *ServiceImpl) Create(ctx context.Context, blueprint *Blueprint) error {
	// Simple validation
	if blueprint.Module == "" || blueprint.Name == "" {
		return fmt.Errorf("module and name are required")
	}
	return s.repo.Create(ctx, blueprint)
}

func (s *ServiceImpl) Update(ctx context.Context, id string, blueprint *Blueprint) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("blueprint not found")
	}

	blueprint.ID = existing.ID
	blueprint.TenantID = existing.TenantID
	blueprint.CreatedAt = existing.CreatedAt

	return s.repo.Update(ctx, blueprint)
}

func (s *ServiceImpl) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ServiceImpl) GetByID(ctx context.Context, id string) (*Blueprint, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *ServiceImpl) List(ctx context.Context, filter BlueprintFilter) ([]Blueprint, error) {
	return s.repo.List(ctx, filter)
}

func (s *ServiceImpl) GetActiveByModule(ctx context.Context, module string) (*Blueprint, error) {
	return s.repo.FindActiveByModule(ctx, module)
}

// ExecuteTransition handles the state change and associated actions
func (s *ServiceImpl) ExecuteTransition(ctx context.Context, blueprintID string, transitionID string, recordID string, inputData map[string]interface{}) error {
	// 1. Fetch Blueprint
	bp, err := s.repo.FindByID(ctx, blueprintID)
	if err != nil {
		return err
	}
	if bp == nil {
		return fmt.Errorf("blueprint not found")
	}

	// 2. Find Transition
	var transition *Transition
	for _, t := range bp.Transitions {
		if t.ID == transitionID {
			val := t
			transition = &val
			break
		}
	}
	if transition == nil {
		return fmt.Errorf("transition not found")
	}

	// 3. Fetch Record
	rec, err := s.recordRepo.Get(ctx, bp.Module, recordID)
	if err != nil {
		return fmt.Errorf("failed to fetch record: %w", err)
	}
	if rec == nil {
		return fmt.Errorf("record not found")
	}

	// 4. Validate Conditions (Criteria)
	if !s.evaluateCriteria(rec, transition.Criteria) {
		return fmt.Errorf("transition criteria not met")
	}

	// 5. Run BEFORE Actions
	if err := s.executeActions(ctx, transition.Before, bp.Module, rec); err != nil {
		return fmt.Errorf("failed to run before actions: %w", err)
	}

	// 6. Update Record State (DURING)
	// Update the TargetField to 'ToState'
	if bp.TargetField != "" {
		updateData := map[string]interface{}{
			bp.TargetField: transition.ToState,
			"updated_at":   time.Now(),
		}
		// If inputData has generic updates, merge them?
		// For now, prioritize state change.
		if err := s.recordRepo.Update(ctx, bp.Module, recordID, updateData); err != nil {
			return fmt.Errorf("failed to update record state: %w", err)
		}
		// Refresh record data for subsequent actions
		rec[bp.TargetField] = transition.ToState
	}

	// 7. Run DURING Actions
	if err := s.executeActions(ctx, transition.During, bp.Module, rec); err != nil {
		return fmt.Errorf("failed to run during actions: %w", err)
	}

	// 8. Run AFTER Actions (Non-blocking usually, but here we run sequentially)
	if err := s.executeActions(ctx, transition.After, bp.Module, rec); err != nil {
		// Log but don't fail?
		fmt.Printf("Warning: failed to run after actions: %v\n", err)
	}

	return nil
}

func (s *ServiceImpl) ValidateTransition(ctx context.Context, blueprintID string, transitionID string, recordID string) (bool, error) {
	// Implementation for specific validation check without executing
	return true, nil
}

func (s *ServiceImpl) executeActions(ctx context.Context, actions []BlueprintAction, moduleName string, rec map[string]interface{}) error {
	for _, action := range actions {
		ruleAction := s.mapToRuleAction(action)
		if err := s.executor.ExecuteAction(ctx, ruleAction, moduleName, rec); err != nil {
			return err
		}
	}
	return nil
}

func (s *ServiceImpl) mapToRuleAction(action BlueprintAction) automation.RuleAction {
	var ruleType automation.ActionType
	switch action.Type {
	case ActionTypeEmail:
		ruleType = automation.ActionSendEmail
	case ActionTypeFieldUpdate:
		ruleType = automation.ActionUpdateField
	case ActionTypeWebhook:
		ruleType = automation.ActionWebhook
	case ActionTypeScript:
		ruleType = automation.ActionRunScript
	default:
		// Fallback or unknown
		ruleType = automation.ActionType(action.Type)
	}

	return automation.RuleAction{
		Type:   ruleType,
		Config: action.Config,
	}
}

func (s *ServiceImpl) evaluateCriteria(rec map[string]interface{}, criteria []common_models.Filter) bool {
	if len(criteria) == 0 {
		return true
	}

	for _, filter := range criteria {
		val, exists := rec[filter.Field]
		if !exists {
			return false // Field missing
		}

		if !evaluateCondition(val, filter.Operator, filter.Value) {
			return false
		}
	}
	return true
}

func evaluateCondition(actual interface{}, operator string, expected interface{}) bool {
	sActual := fmt.Sprintf("%v", actual)
	sExpected := fmt.Sprintf("%v", expected)

	switch operator {
	case "eq", "equals":
		return sActual == sExpected
	case "ne", "not_equals":
		return sActual != sExpected
	case "contains":
		return strings.Contains(sActual, sExpected)
	// Add more operators as needed (gt, lt, etc.)
	default:
		return false
	}
}
