package service

import (
	"context"
	"errors"
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
	ListWorkflows(ctx context.Context) ([]models.ApprovalWorkflow, error)

	// Approval Actions
	ApproveRecord(ctx context.Context, moduleName string, recordID string, actorID string, comment string) error
	RejectRecord(ctx context.Context, moduleName string, recordID string, actorID string, comment string) error

	// Helper to check if a user can approve the current step
	CanApprove(ctx context.Context, moduleName string, recordID string, userID string, userRoleIDs []string) (bool, error)

	// Helper to initialize approval state for a new record
	InitializeApproval(ctx context.Context, moduleName string) (*models.ApprovalRecordState, error)
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
	// Check if workflow already exists for module
	existing, err := s.Repo.GetByModuleID(ctx, workflow.ModuleID)
	if err == nil && existing != nil && existing.Active && workflow.Active {
		return errors.New("active workflow already exists for this module")
	}

	if workflow.ID.IsZero() {
		workflow.ID = primitive.NewObjectID()
	}
	workflow.CreatedAt = time.Now()
	workflow.UpdatedAt = time.Now()

	return s.Repo.Create(ctx, workflow)
}

func (s *ApprovalServiceImpl) GetWorkflowByModule(ctx context.Context, moduleID string) (*models.ApprovalWorkflow, error) {
	return s.Repo.GetByModuleID(ctx, moduleID)
}

func (s *ApprovalServiceImpl) ListWorkflows(ctx context.Context) ([]models.ApprovalWorkflow, error) {
	return s.Repo.List(ctx)
}

func (s *ApprovalServiceImpl) InitializeApproval(ctx context.Context, moduleName string) (*models.ApprovalRecordState, error) {
	// 1. Get Module ID by Name
	module, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return nil, err
	}

	// 2. Check for Active Workflow
	workflow, err := s.GetWorkflowByModule(ctx, module.ID.Hex())
	if err != nil || workflow == nil {
		return nil, nil // No workflow, approval not required
	}

	// 3. Return Initial State
	return &models.ApprovalRecordState{
		Status:      models.ApprovalStatusPending,
		CurrentStep: 0,
		WorkflowID:  workflow.ID.Hex(),
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
