package service

import (
	"context"
	"errors"
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"slices"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ApprovalService interface {
	CreateWorkflow(ctx context.Context, workflow models.ApprovalWorkflow) error
	GetWorkflowByModule(ctx context.Context, moduleID string) (*models.ApprovalWorkflow, error)
	GetWorkflowByID(ctx context.Context, id string) (*models.ApprovalWorkflow, error)
	ListWorkflows(ctx context.Context) ([]models.ApprovalWorkflow, error)
	UpdateWorkflow(ctx context.Context, id string, workflow models.ApprovalWorkflow) error
	DeleteWorkflow(ctx context.Context, id string) error

	// Approval Actions
	ApproveRecord(ctx context.Context, moduleName string, recordID string, actorID string, comment string) error
	RejectRecord(ctx context.Context, moduleName string, recordID string, actorID string, comment string) error

	// Helper to check if a user can approve the current step
	CanApprove(ctx context.Context, moduleName string, recordID string, userID string, userRoleIDs []string) (bool, error)

	// Helper to initialize approval state for a new record
	InitializeApproval(ctx context.Context, moduleName string, record map[string]interface{}) (*models.ApprovalRecordState, error)
}

type ApprovalServiceImpl struct {
	Repo         repository.ApprovalRepository
	RecordRepo   repository.RecordRepository
	ModuleRepo   repository.ModuleRepository
	UserRepo     repository.UserRepository
	AuditService AuditService
}

func NewApprovalService(
	repo repository.ApprovalRepository,
	recordRepo repository.RecordRepository,
	moduleRepo repository.ModuleRepository,
	userRepo repository.UserRepository,
	auditService AuditService,
) ApprovalService {
	return &ApprovalServiceImpl{
		Repo:         repo,
		RecordRepo:   recordRepo,
		ModuleRepo:   moduleRepo,
		UserRepo:     userRepo,
		AuditService: auditService,
	}
}

func (s *ApprovalServiceImpl) CreateWorkflow(ctx context.Context, workflow models.ApprovalWorkflow) error {
	// Validate overlaps
	if err := s.validateWorkflowOverlaps(ctx, workflow); err != nil {
		return err
	}

	if workflow.ID.IsZero() {
		workflow.ID = primitive.NewObjectID()
	}
	workflow.CreatedAt = time.Now()
	workflow.UpdatedAt = time.Now()

	return s.Repo.Create(ctx, workflow)
}

func (s *ApprovalServiceImpl) UpdateWorkflow(ctx context.Context, id string, workflow models.ApprovalWorkflow) error {
	// Ensure ID is set for validation check exclusion
	workflow.ID, _ = primitive.ObjectIDFromHex(id)

	if err := s.validateWorkflowOverlaps(ctx, workflow); err != nil {
		return err
	}

	workflow.UpdatedAt = time.Now()
	return s.Repo.Update(ctx, id, workflow)
}

func (s *ApprovalServiceImpl) validateWorkflowOverlaps(ctx context.Context, workflow models.ApprovalWorkflow) error {
	if !workflow.Active {
		return nil // Inactive workflows don't overlap
	}

	existingWorkflows, err := s.Repo.ListActiveByModuleID(ctx, workflow.ModuleID)
	if err != nil {
		return err
	}

	for _, ef := range existingWorkflows {
		// Skip self (for Update)
		if ef.ID == workflow.ID {
			continue
		}

		// Check for multiple defaults
		if len(workflow.Criteria) == 0 && len(ef.Criteria) == 0 {
			return errors.New("a default workflow (no criteria) already exists for this module")
		}

		// Check for identical criteria (Exact Match)
		// This is a basic check.
		if len(workflow.Criteria) > 0 && len(ef.Criteria) == len(workflow.Criteria) {
			matchCount := 0
			for _, c1 := range workflow.Criteria {
				for _, c2 := range ef.Criteria {
					// Compare Field, Operator, Value
					if c1.Field == c2.Field && c1.Operator == c2.Operator && c1.Value == c2.Value {
						matchCount++
						break
					}
				}
			}
			if matchCount == len(workflow.Criteria) {
				return errors.New("a workflow with identical criteria already exists")
			}
		}
	}
	return nil
}

func (s *ApprovalServiceImpl) DeleteWorkflow(ctx context.Context, id string) error {
	return s.Repo.Delete(ctx, id)
}

func (s *ApprovalServiceImpl) GetWorkflowByModule(ctx context.Context, moduleID string) (*models.ApprovalWorkflow, error) {
	return s.Repo.GetByModuleID(ctx, moduleID)
}

func (s *ApprovalServiceImpl) GetWorkflowByID(ctx context.Context, id string) (*models.ApprovalWorkflow, error) {
	return s.Repo.GetByID(ctx, id)
}

func (s *ApprovalServiceImpl) ListWorkflows(ctx context.Context) ([]models.ApprovalWorkflow, error) {
	return s.Repo.List(ctx)
}

func (s *ApprovalServiceImpl) InitializeApproval(ctx context.Context, moduleName string, record map[string]interface{}) (*models.ApprovalRecordState, error) {
	// 1. Get Module ID by Name
	module, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return nil, err
	}

	// 2. Refresh: Fetch ALL Active Workflows for Module
	workflows, err := s.Repo.ListActiveByModuleID(ctx, module.ID.Hex())
	if err != nil || len(workflows) == 0 {
		return nil, nil // No workflow, approval not required
	}

	// SORT by Priority (0 is highest/first)
	slices.SortFunc(workflows, func(a, b models.ApprovalWorkflow) int {
		if a.Priority != b.Priority {
			return a.Priority - b.Priority
		}
		// Tie-breaker: CreatedAt (Older first?)
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		return 1
	})

	// 3. Find Matching Workflow
	var matchedWorkflow *models.ApprovalWorkflow

	// AutomationService has evaluateConditions logic.
	// To avoid duplication, I should maybe move evaluateConditions to a shared helper or utils.
	// For now, I will inline a simple checker or call Automation logic?
	// AutomationService logic is private `evaluateConditions`.
	// Let's implement a simple matcher here locally.

	for _, wf := range workflows {
		if len(wf.Criteria) == 0 {
			// Default workflow logic?
			// If multiple workflows exist, one sans-criteria might be "default"
			// Let's deprioritize it if others match? Or just take first match.
			// Usually specific criteria > generic.
			// For now: First match wins.
			matchedWorkflow = &wf
			break
		}

		match := true
		for _, cond := range wf.Criteria {
			val, exists := record[cond.Field]
			if !exists {
				match = false
				break
			}

			// Simple string comparison for MVP
			// Reuse Automation logic ideally.
			strVal := fmt.Sprintf("%v", val)
			strCond := fmt.Sprintf("%v", cond.Value)

			if cond.Operator == models.OperatorEquals && strVal != strCond {
				match = false
				break
			}
			// Add other operators as needed
		}
		if match {
			matchedWorkflow = &wf
			break
		}
	}

	if matchedWorkflow == nil {
		return nil, nil // No matching workflow
	}

	// 3. Return Initial State
	return &models.ApprovalRecordState{
		Status:      models.ApprovalStatusPending,
		CurrentStep: 0,
		WorkflowID:  matchedWorkflow.ID.Hex(),
		History:     []models.ApprovalHistory{},
	}, nil
}

func (s *ApprovalServiceImpl) ApproveRecord(ctx context.Context, moduleName string, recordID string, actorID string, comment string) error {
	// 1. Get Record & Current State
	record, err := s.RecordRepo.Get(ctx, moduleName, recordID)
	if err != nil {
		return err
	}

	state := s.extractApprovalState(record)
	if state == nil || state.Status != models.ApprovalStatusPending {
		return errors.New("record is not pending approval")
	}

	// 2. Get Workflow
	workflow, err := s.Repo.GetByID(ctx, state.WorkflowID)
	if err != nil {
		return err
	}

	if state.CurrentStep >= len(workflow.Steps) {
		return errors.New("invalid approval step")
	}

	currentStep := workflow.Steps[state.CurrentStep]

	// 3. Update State
	// Log History
	history := models.ApprovalHistory{
		StepName:  currentStep.Name,
		ActorID:   actorID,
		Action:    models.ApprovalStatusApproved,
		Comment:   comment,
		Timestamp: time.Now(),
	}
	state.History = append(state.History, history)

	// Advance Step
	if state.CurrentStep < len(workflow.Steps)-1 {
		state.CurrentStep++
	} else {
		// Final Step Approved
		state.Status = models.ApprovalStatusApproved
	}

	// 4. Save Record
	data := map[string]interface{}{
		"_approval": state,
	}

	err = s.RecordRepo.Update(ctx, moduleName, recordID, data)
	if err != nil {
		return err
	}

	// 5. Log Audit Entry
	changes := map[string]models.Change{
		"_approval": {
			Old: "pending",
			New: "approved",
		},
		"approval_comment": {
			Old: nil,
			New: comment,
		},
	}
	_ = s.AuditService.LogChange(ctx, models.AuditActionUpdate, moduleName, recordID, changes)

	return nil
}

func (s *ApprovalServiceImpl) RejectRecord(ctx context.Context, moduleName string, recordID string, actorID string, comment string) error {
	record, err := s.RecordRepo.Get(ctx, moduleName, recordID)
	if err != nil {
		return err
	}

	state := s.extractApprovalState(record)
	if state == nil || state.Status != models.ApprovalStatusPending {
		return errors.New("record is not pending approval")
	}

	workflow, err := s.Repo.GetByID(ctx, state.WorkflowID)
	if err != nil {
		return err
	}

	currentStep := workflow.Steps[state.CurrentStep]

	// Log History
	history := models.ApprovalHistory{
		StepName:  currentStep.Name,
		ActorID:   actorID,
		Action:    models.ApprovalStatusRejected,
		Comment:   comment,
		Timestamp: time.Now(),
	}
	state.History = append(state.History, history)

	// Set Status to Rejected
	state.Status = models.ApprovalStatusRejected

	data := map[string]interface{}{
		"_approval": state,
	}
	err = s.RecordRepo.Update(ctx, moduleName, recordID, data)
	if err != nil {
		return err
	}

	// 5. Log Audit Entry
	changes := map[string]models.Change{
		"_approval": {
			Old: "pending",
			New: "rejected",
		},
		"rejection_comment": {
			Old: nil,
			New: comment,
		},
	}
	_ = s.AuditService.LogChange(ctx, models.AuditActionUpdate, moduleName, recordID, changes)

	return nil
}

func (s *ApprovalServiceImpl) CanApprove(ctx context.Context, moduleName string, recordID string, userID string, userRoleIDs []string) (bool, error) {
	record, err := s.RecordRepo.Get(ctx, moduleName, recordID)
	if err != nil {
		return false, err
	}

	state := s.extractApprovalState(record)
	if state == nil || state.Status != models.ApprovalStatusPending {
		return false, nil
	}

	workflow, err := s.Repo.GetByID(ctx, state.WorkflowID)
	if err != nil {
		return false, err
	}

	if state.CurrentStep >= len(workflow.Steps) {
		return false, nil
	}

	step := workflow.Steps[state.CurrentStep]

	// Check if user is in AllowedUsers
	if slices.Contains(step.ApproverUsers, userID) {
		return true, nil
	}

	// Check if user has one of the AllowedRoles
	for _, roleID := range userRoleIDs {
		if slices.Contains(step.ApproverRoles, roleID) {
			return true, nil
		}
	}

	return false, nil
}

// Helper to extract approval state from map
func (s *ApprovalServiceImpl) extractApprovalState(record map[string]any) *models.ApprovalRecordState {
	if val, ok := record["_approval"]; ok {
		// Use bson conversion if needed, or straightforward casting if type is preserved
		// Usually mongo driver unmarshals into primitive.M or map[string]interface{}
		// We might need to marshal/unmarshal to struct

		// For simplicity/robustness, assuming we can cast or re-decode
		// Since we're in service, dealing with raw maps is tricky.
		// Let's rely on bson library to convert map to struct
		var state models.ApprovalRecordState

		// Quick and dirty conversion via bson roundtrip
		bytes, _ := bson.Marshal(val)
		bson.Unmarshal(bytes, &state)
		return &state
	}
	return nil
}
