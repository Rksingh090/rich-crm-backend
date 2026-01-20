package approval

import (
	"context"
	"errors"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/core/audit"
	"go-crm/internal/core/role"
	"go-crm/internal/core/user"
	"go-crm/internal/features/module"
	"go-crm/internal/features/record"
	"slices"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ApprovalService interface {
	CreateApprovalProcess(ctx context.Context, process ApprovalProcess, userID primitive.ObjectID) (*ApprovalProcess, error)
	GetApprovalProcessByModule(ctx context.Context, moduleID string, userID primitive.ObjectID) (*ApprovalProcess, error)
	GetApprovalProcessByID(ctx context.Context, id string, userID primitive.ObjectID) (*ApprovalProcess, error)
	ListApprovalProcesses(ctx context.Context, userID primitive.ObjectID) ([]ApprovalProcess, error)
	UpdateApprovalProcess(ctx context.Context, id string, process ApprovalProcess, userID primitive.ObjectID) error
	DeleteApprovalProcess(ctx context.Context, id string, userID primitive.ObjectID) error

	// Approval Actions
	ApproveRecord(ctx context.Context, moduleName string, recordID string, actorID string, comment string) error
	RejectRecord(ctx context.Context, moduleName string, recordID string, actorID string, comment string) error

	// Helper to check if a user can approve the current step
	CanApprove(ctx context.Context, moduleName string, recordID string, userID string, userRoleIDs []string) (bool, error)

	// Helper to initialize approval state for a new record
	InitializeApproval(ctx context.Context, moduleName string, record map[string]any) (*common_models.ApprovalRecordState, error)
}

type ApprovalServiceImpl struct {
	Repo         ApprovalRepository
	RecordRepo   record.RecordRepository
	ModuleRepo   module.ModuleRepository
	UserRepo     user.UserRepository
	AuditService audit.AuditService
	RoleService  role.RoleService
}

func NewApprovalService(
	repo ApprovalRepository,
	recordRepo record.RecordRepository,
	moduleRepo module.ModuleRepository,
	userRepo user.UserRepository,
	auditService audit.AuditService,
	roleService role.RoleService,
) *ApprovalServiceImpl {
	return &ApprovalServiceImpl{
		Repo:         repo,
		RecordRepo:   recordRepo,
		ModuleRepo:   moduleRepo,
		UserRepo:     userRepo,
		AuditService: auditService,
		RoleService:  roleService,
	}
}

func (s *ApprovalServiceImpl) CreateApprovalProcess(ctx context.Context, process ApprovalProcess, userID primitive.ObjectID) (*ApprovalProcess, error) {
	// Permission Check
	if !userID.IsZero() {
		allowed, err := s.RoleService.CheckPermission(ctx, userID, "settings_approval_processes", "create")
		if err != nil || !allowed {
			return nil, errors.New("access denied")
		}
	}

	if err := s.validateApprovalProcessOverlaps(ctx, process); err != nil {
		return nil, err
	}

	if process.ID.IsZero() {
		process.ID = primitive.NewObjectID()
	}
	process.CreatedAt = time.Now()
	process.UpdatedAt = time.Now()

	if err := s.Repo.Create(ctx, &process); err != nil {
		return nil, err
	}

	return &process, nil
}

func (s *ApprovalServiceImpl) UpdateApprovalProcess(ctx context.Context, id string, process ApprovalProcess, userID primitive.ObjectID) error {
	// Permission Check
	if !userID.IsZero() {
		allowed, err := s.RoleService.CheckPermission(ctx, userID, "settings_approval_processes", "update")
		if err != nil || !allowed {
			return errors.New("access denied")
		}
	}

	process.ID, _ = primitive.ObjectIDFromHex(id)

	if err := s.validateApprovalProcessOverlaps(ctx, process); err != nil {
		return err
	}

	process.UpdatedAt = time.Now()
	return s.Repo.Update(ctx, id, process)
}

func (s *ApprovalServiceImpl) validateApprovalProcessOverlaps(ctx context.Context, process ApprovalProcess) error {
	if !process.Active {
		return nil
	}

	existingProcesses, err := s.Repo.ListActiveByModuleID(ctx, process.ModuleID)
	if err != nil {
		return err
	}

	for _, ef := range existingProcesses {
		if ef.ID == process.ID {
			continue
		}

		if len(process.Criteria) == 0 && len(ef.Criteria) == 0 {
			return errors.New("a default approval process (no criteria) already exists for this module")
		}

		if len(process.Criteria) > 0 && len(ef.Criteria) == len(process.Criteria) {
			matchCount := 0
			for _, c1 := range process.Criteria {
				for _, c2 := range ef.Criteria {
					if c1.Field == c2.Field && c1.Operator == c2.Operator && c1.Value == c2.Value {
						matchCount++
						break
					}
				}
			}
			if matchCount == len(process.Criteria) {
				return errors.New("an approval process with identical criteria already exists")
			}
		}
	}
	return nil
}

func (s *ApprovalServiceImpl) DeleteApprovalProcess(ctx context.Context, id string, userID primitive.ObjectID) error {
	// Permission Check
	if !userID.IsZero() {
		allowed, err := s.RoleService.CheckPermission(ctx, userID, "settings_approval_processes", "delete")
		if err != nil || !allowed {
			return errors.New("access denied")
		}
	}
	return s.Repo.Delete(ctx, id)
}

func (s *ApprovalServiceImpl) GetApprovalProcessByModule(ctx context.Context, moduleID string, userID primitive.ObjectID) (*ApprovalProcess, error) {
	return s.Repo.GetByModuleID(ctx, moduleID)
}

func (s *ApprovalServiceImpl) GetApprovalProcessByID(ctx context.Context, id string, userID primitive.ObjectID) (*ApprovalProcess, error) {
	return s.Repo.GetByID(ctx, id)
}

func (s *ApprovalServiceImpl) ListApprovalProcesses(ctx context.Context, userID primitive.ObjectID) ([]ApprovalProcess, error) {
	// Permission Check
	if !userID.IsZero() {
		allowed, err := s.RoleService.CheckPermission(ctx, userID, "settings_approval_processes", "read")
		if err != nil || !allowed {
			return nil, errors.New("access denied")
		}
	}
	return s.Repo.List(ctx)
}

func (s *ApprovalServiceImpl) InitializeApproval(ctx context.Context, moduleName string, rec map[string]any) (*common_models.ApprovalRecordState, error) {
	mod, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return nil, err
	}

	processes, err := s.Repo.ListActiveByModuleID(ctx, mod.ID.Hex())
	if err != nil || len(processes) == 0 {
		return nil, nil
	}

	slices.SortFunc(processes, func(a, b ApprovalProcess) int {
		if a.Priority != b.Priority {
			return a.Priority - b.Priority
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		return 1
	})

	var matchedProcess *ApprovalProcess

	for _, wf := range processes {
		if len(wf.Criteria) == 0 {
			matchedProcess = &wf
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
			matchedProcess = &wf
			break
		}
	}

	if matchedProcess == nil {
		return nil, nil
	}

	return &common_models.ApprovalRecordState{
		Status:      common_models.ApprovalStatusPending,
		CurrentStep: 0,
		ProcessID:   matchedProcess.ID.Hex(),
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

	process, err := s.Repo.GetByID(ctx, state.ProcessID)
	if err != nil {
		return err
	}

	if state.CurrentStep >= len(process.Steps) {
		return errors.New("invalid approval step")
	}

	currentStep := process.Steps[state.CurrentStep]

	history := common_models.ApprovalHistory{
		StepName:  currentStep.Name,
		ActorID:   actorID,
		Action:    common_models.ApprovalStatusApproved,
		Comment:   comment,
		Timestamp: time.Now(),
	}
	state.History = append(state.History, history)

	if state.CurrentStep < len(process.Steps)-1 {
		state.CurrentStep++
	} else {
		state.Status = common_models.ApprovalStatusApproved
	}

	data := map[string]any{
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

	process, err := s.Repo.GetByID(ctx, state.ProcessID)
	if err != nil {
		return err
	}

	currentStep := process.Steps[state.CurrentStep]

	history := common_models.ApprovalHistory{
		StepName:  currentStep.Name,
		ActorID:   actorID,
		Action:    common_models.ApprovalStatusRejected,
		Comment:   comment,
		Timestamp: time.Now(),
	}
	state.History = append(state.History, history)

	state.Status = common_models.ApprovalStatusRejected

	data := map[string]any{
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

	process, err := s.Repo.GetByID(ctx, state.ProcessID)
	if err != nil {
		return false, err
	}

	if state.CurrentStep >= len(process.Steps) {
		return false, nil
	}

	step := process.Steps[state.CurrentStep]

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
