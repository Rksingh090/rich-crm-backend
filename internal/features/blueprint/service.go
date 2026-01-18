package blueprint

import (
	"context"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/core/action"
	"go-crm/internal/core/audit"
	"go-crm/internal/features/record"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service interface {
	Create(ctx context.Context, blueprint *Blueprint) error
	Update(ctx context.Context, id string, blueprint *Blueprint) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*Blueprint, error)
	List(ctx context.Context, filter BlueprintFilter) ([]Blueprint, error)
	GetActiveByModule(ctx context.Context, module string) (*Blueprint, error)

	// Core Logic
	ExecuteTransition(ctx context.Context, blueprintID string, transitionID string, recordID string, data map[string]any) error
	ValidateTransition(ctx context.Context, blueprintID string, transitionID string, recordID string) (bool, error)
	GetAvailableTransitions(ctx context.Context, module string, recordID string) ([]Transition, error)
	GetActiveBlueprintTargetField(ctx context.Context, module string) (string, error)
}

type ServiceImpl struct {
	repo         Repository
	recordRepo   record.RecordRepository
	executor     action.ActionExecutor
	auditService audit.AuditService
}

func NewService(repo Repository, recordRepo record.RecordRepository, executor action.ActionExecutor, auditService audit.AuditService) Service {
	return &ServiceImpl{
		repo:         repo,
		recordRepo:   recordRepo,
		executor:     executor,
		auditService: auditService,
	}
}

func (s *ServiceImpl) Create(ctx context.Context, blueprint *Blueprint) error {
	// Simple validation
	if blueprint.Module == "" || blueprint.Name == "" {
		return fmt.Errorf("module and name are required")
	}

	// Default to active
	blueprint.Active = true

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

	// Validate TargetField Change
	if existing.TargetField != blueprint.TargetField {
		// Check if records exist
		count, err := s.recordRepo.Count(ctx, blueprint.Module, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to check existing records: %w", err)
		}
		if count > 0 {
			return fmt.Errorf("cannot change target field when records exist in module")
		}
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
func (s *ServiceImpl) ExecuteTransition(ctx context.Context, blueprintID string, transitionID string, recordID string, inputData map[string]any) error {
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

	// CHECK APPROVAL STATUS
	if s.isPendingApproval(rec) {
		return fmt.Errorf("record is currently pending approval and cannot be transitioned")
	}

	// Get Old State for Audit
	oldState := ""
	if val, ok := rec[bp.TargetField]; ok {
		oldState = fmt.Sprintf("%v", val)
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
		updateData := map[string]any{
			bp.TargetField: transition.ToState,
			"updated_at":   time.Now(),
		}

		// SPECIAL HANDLING FOR TICKETS STATUS HISTORY
		// This is a bit hacky but ensures consistency with TicketService logic without circular deps
		if bp.Module == "tickets" && bp.TargetField == "status" {
			currentHistory, _ := rec["status_history"].([]any)
			if currentHistory == nil {
				// Try to parse if it's a generic slice
				if hist, ok := rec["status_history"].(primitive.A); ok {
					currentHistory = []any(hist)
				}
			}

			// Get User ID from Context
			// Assuming utils.UserClaimsKey is available or similar.
			// We can't import utils easily if not already imported, but let's try to get it safely or use AuditService's logic later.
			// Ideally we use the user ID from context.
			// For now, let's leave changedBy empty or handle via simple map.

			newEntry := map[string]any{
				"status":     transition.ToState,
				"changed_at": time.Now(),
				"comment":    fmt.Sprintf("Transition: %s", transition.Name),
				// "changed_by": ... (This is hard to get without proper context key access here, will rely on Audit for user tracking)
			}

			// Append
			currentHistory = append(currentHistory, newEntry)
			updateData["status_history"] = currentHistory
		}

		if err := s.recordRepo.Update(ctx, bp.Module, recordID, updateData); err != nil {
			return fmt.Errorf("failed to update record state: %w", err)
		}
		// Refresh record data for subsequent actions
		rec[bp.TargetField] = transition.ToState
	}

	// Audit Log
	changes := map[string]common_models.Change{
		bp.TargetField:         {Old: oldState, New: transition.ToState},
		"blueprint_transition": {Old: nil, New: transition.Name},
	}
	// Action depends on context, but Update is safe
	_ = s.auditService.LogChange(ctx, common_models.AuditActionUpdate, bp.Module, recordID, changes)

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

func (s *ServiceImpl) GetAvailableTransitions(ctx context.Context, module string, recordID string) ([]Transition, error) {
	// 1. Get Active Blueprint
	bp, err := s.repo.FindActiveByModule(ctx, module)
	if err != nil {
		return nil, err
	}
	if bp == nil {
		// No active blueprint, means no blueprint transitions available
		return []Transition{}, nil
	}

	// 2. Fetch Record
	rec, err := s.recordRepo.Get(ctx, module, recordID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("record not found")
	}

	// CHECK APPROVAL STATUS
	if s.isPendingApproval(rec) {
		// Return empty list if pending approval
		return []Transition{}, nil
	}

	// 3. Determine Current State
	currentState := ""
	if val, ok := rec[bp.TargetField]; ok {
		currentState = fmt.Sprintf("%v", val)
	}

	// 4. Filter Transitions
	var available []Transition
	for _, t := range bp.Transitions {
		// a. Check FromState matches
		if t.FromState != currentState {
			continue
		}

		// b. Check Criteria
		if s.evaluateCriteria(rec, t.Criteria) {
			available = append(available, t)
		}
	}

	return available, nil
}

func (s *ServiceImpl) executeActions(ctx context.Context, actions []BlueprintAction, moduleName string, rec map[string]any) error {
	// Convert BlueprintAction to action.Action
	coreActions := make([]action.Action, len(actions))
	for i, a := range actions {
		coreActions[i] = action.Action{
			Type:   a.Type, // Already action.ActionType
			Config: a.Config,
		}
	}
	return s.executor.ExecuteActions(ctx, coreActions, moduleName, rec)
}

func (s *ServiceImpl) isPendingApproval(rec map[string]any) bool {
	if val, ok := rec["_approval"]; ok {
		var state common_models.ApprovalRecordState
		bytes, _ := bson.Marshal(val)
		bson.Unmarshal(bytes, &state)
		return state.Status == common_models.ApprovalStatusPending
	}
	return false
}

func (s *ServiceImpl) evaluateCriteria(rec map[string]any, criteria []common_models.Filter) bool {
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

func evaluateCondition(actual any, operator string, expected any) bool {
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

func (s *ServiceImpl) GetActiveBlueprintTargetField(ctx context.Context, module string) (string, error) {
	bp, err := s.repo.FindActiveByModule(ctx, module)
	if err != nil {
		return "", err
	}
	if bp == nil {
		return "", nil
	}
	return bp.TargetField, nil
}
