package approval

import (
	"context"
	"errors"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/module"
	"go-crm/internal/features/record"
	"go-crm/internal/features/user"
	"slices"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ApprovalService interface {
	CreateWorkflow(ctx context.Context, workflow ApprovalWorkflow) error
	GetWorkflowByModule(ctx context.Context, moduleID string) (*ApprovalWorkflow, error)
	GetWorkflowByID(ctx context.Context, id string) (*ApprovalWorkflow, error)
	ListWorkflows(ctx context.Context) ([]ApprovalWorkflow, error)
	UpdateWorkflow(ctx context.Context, id string, workflow ApprovalWorkflow) error
	DeleteWorkflow(ctx context.Context, id string) error

	// Approval Actions
	ApproveRecord(ctx context.Context, moduleName string, recordID string, actorID string, comment string) error
	RejectRecord(ctx context.Context, moduleName string, recordID string, actorID string, comment string) error

	// Helper to check if a user can approve the current step
	CanApprove(ctx context.Context, moduleName string, recordID string, userID string, userRoleIDs []string) (bool, error)

	// Helper to initialize approval state for a new record
	InitializeApproval(ctx context.Context, moduleName string, record map[string]interface{}) (*common_models.ApprovalRecordState, error)
}

type ApprovalServiceImpl struct {
	Repo         ApprovalRepository
	RecordRepo   record.RecordRepository
	ModuleRepo   module.ModuleRepository
	UserRepo     user.UserRepository
	AuditService audit.AuditService
}

func NewApprovalService(
	repo ApprovalRepository,
	recordRepo record.RecordRepository,
	moduleRepo module.ModuleRepository,
	userRepo user.UserRepository,
	auditService audit.AuditService,
) ApprovalService {
	return &ApprovalServiceImpl{
		Repo:         repo,
		RecordRepo:   recordRepo,
		ModuleRepo:   moduleRepo,
		UserRepo:     userRepo,
		AuditService: auditService,
	}
}

func (s *ApprovalServiceImpl) CreateWorkflow(ctx context.Context, workflow ApprovalWorkflow) error {
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

func (s *ApprovalServiceImpl) UpdateWorkflow(ctx context.Context, id string, workflow ApprovalWorkflow) error {
	workflow.ID, _ = primitive.ObjectIDFromHex(id)

	if err := s.validateWorkflowOverlaps(ctx, workflow); err != nil {
		return err
	}

	workflow.UpdatedAt = time.Now()
	return s.Repo.Update(ctx, id, workflow)
}

func (s *ApprovalServiceImpl) validateWorkflowOverlaps(ctx context.Context, workflow ApprovalWorkflow) error {
	if !workflow.Active {
		return nil
	}

	existingWorkflows, err := s.Repo.ListActiveByModuleID(ctx, workflow.ModuleID)
	if err != nil {
		return err
	}

	for _, ef := range existingWorkflows {
		if ef.ID == workflow.ID {
			continue
		}

		if len(workflow.Criteria) == 0 && len(ef.Criteria) == 0 {
			return errors.New("a default workflow (no criteria) already exists for this module")
		}

		if len(workflow.Criteria) > 0 && len(ef.Criteria) == len(workflow.Criteria) {
			matchCount := 0
			for _, c1 := range workflow.Criteria {
				for _, c2 := range ef.Criteria {
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

func (s *ApprovalServiceImpl) GetWorkflowByModule(ctx context.Context, moduleID string) (*ApprovalWorkflow, error) {
	return s.Repo.GetByModuleID(ctx, moduleID)
}

func (s *ApprovalServiceImpl) GetWorkflowByID(ctx context.Context, id string) (*ApprovalWorkflow, error) {
	return s.Repo.GetByID(ctx, id)
}

func (s *ApprovalServiceImpl) ListWorkflows(ctx context.Context) ([]ApprovalWorkflow, error) {
	return s.Repo.List(ctx)
}

func (s *ApprovalServiceImpl) InitializeApproval(ctx context.Context, moduleName string, rec map[string]interface{}) (*common_models.ApprovalRecordState, error) {
	mod, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return nil, err
	}

	workflows, err := s.Repo.ListActiveByModuleID(ctx, mod.ID.Hex())
	if err != nil || len(workflows) == 0 {
		return nil, nil
	}

	slices.SortFunc(workflows, func(a, b ApprovalWorkflow) int {
		if a.Priority != b.Priority {
			return a.Priority - b.Priority
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		return 1
	})

	var matchedWorkflow *ApprovalWorkflow

	for _, wf := range workflows {
		if len(wf.Criteria) == 0 {
			matchedWorkflow = &wf
			break
		}

		match := true
		for _, cond := range wf.Criteria {
			val, exists := rec[cond.Field]
			if !exists {
				match = false
				break
			}

			strVal := fmt.Sprintf("%v", val)
			strCond := fmt.Sprintf("%v", cond.Value)

			if cond.Operator == "equals" && strVal != strCond {
				match = false
				break
			}
		}
		if match {
			matchedWorkflow = &wf
			break
		}
	}

	if matchedWorkflow == nil {
		return nil, nil
	}

	return &common_models.ApprovalRecordState{
		Status:      common_models.ApprovalStatusPending,
		CurrentStep: 0,
		WorkflowID:  matchedWorkflow.ID.Hex(),
		History:     []common_models.ApprovalHistory{},
	}, nil
}

func (s *ApprovalServiceImpl) ApproveRecord(ctx context.Context, moduleName string, recordID string, actorID string, comment string) error {
	rec, err := s.RecordRepo.Get(ctx, moduleName, recordID)
	if err != nil {
		return err
	}

	state := s.extractApprovalState(rec)
	if state == nil || state.Status != common_models.ApprovalStatusPending {
		return errors.New("record is not pending approval")
	}

	workflow, err := s.Repo.GetByID(ctx, state.WorkflowID)
	if err != nil {
		return err
	}

	if state.CurrentStep >= len(workflow.Steps) {
		return errors.New("invalid approval step")
	}

	currentStep := workflow.Steps[state.CurrentStep]

	history := common_models.ApprovalHistory{
		StepName:  currentStep.Name,
		ActorID:   actorID,
		Action:    common_models.ApprovalStatusApproved,
		Comment:   comment,
		Timestamp: time.Now(),
	}
	state.History = append(state.History, history)

	if state.CurrentStep < len(workflow.Steps)-1 {
		state.CurrentStep++
	} else {
		state.Status = common_models.ApprovalStatusApproved
	}

	data := map[string]interface{}{
		"_approval": state,
	}

	err = s.RecordRepo.Update(ctx, moduleName, recordID, data)
	if err != nil {
		return err
	}

	changes := map[string]common_models.Change{
		"_approval": {
			Old: "pending",
			New: "approved",
		},
		"approval_comment": {
			Old: nil,
			New: comment,
		},
	}
	_ = s.AuditService.LogChange(ctx, common_models.AuditActionUpdate, moduleName, recordID, changes)

	return nil
}

func (s *ApprovalServiceImpl) RejectRecord(ctx context.Context, moduleName string, recordID string, actorID string, comment string) error {
	rec, err := s.RecordRepo.Get(ctx, moduleName, recordID)
	if err != nil {
		return err
	}

	state := s.extractApprovalState(rec)
	if state == nil || state.Status != common_models.ApprovalStatusPending {
		return errors.New("record is not pending approval")
	}

	workflow, err := s.Repo.GetByID(ctx, state.WorkflowID)
	if err != nil {
		return err
	}

	currentStep := workflow.Steps[state.CurrentStep]

	history := common_models.ApprovalHistory{
		StepName:  currentStep.Name,
		ActorID:   actorID,
		Action:    common_models.ApprovalStatusRejected,
		Comment:   comment,
		Timestamp: time.Now(),
	}
	state.History = append(state.History, history)

	state.Status = common_models.ApprovalStatusRejected

	data := map[string]interface{}{
		"_approval": state,
	}
	err = s.RecordRepo.Update(ctx, moduleName, recordID, data)
	if err != nil {
		return err
	}

	changes := map[string]common_models.Change{
		"_approval": {
			Old: "pending",
			New: "rejected",
		},
		"rejection_comment": {
			Old: nil,
			New: comment,
		},
	}
	_ = s.AuditService.LogChange(ctx, common_models.AuditActionUpdate, moduleName, recordID, changes)

	return nil
}

func (s *ApprovalServiceImpl) CanApprove(ctx context.Context, moduleName string, recordID string, userID string, userRoleIDs []string) (bool, error) {
	rec, err := s.RecordRepo.Get(ctx, moduleName, recordID)
	if err != nil {
		return false, err
	}

	state := s.extractApprovalState(rec)
	if state == nil || state.Status != common_models.ApprovalStatusPending {
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

	if slices.Contains(step.ApproverUsers, userID) {
		return true, nil
	}

	for _, roleID := range userRoleIDs {
		if slices.Contains(step.ApproverRoles, roleID) {
			return true, nil
		}
	}

	return false, nil
}

func (s *ApprovalServiceImpl) extractApprovalState(rec map[string]any) *common_models.ApprovalRecordState {
	if val, ok := rec["_approval"]; ok {
		var state common_models.ApprovalRecordState
		bytes, _ := bson.Marshal(val)
		bson.Unmarshal(bytes, &state)
		return &state
	}
	return nil
}
