package approval

import (
	"context"
	"errors"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/core/action"
	"go-crm/internal/core/audit"
	"go-crm/internal/core/role"
	"go-crm/internal/core/user"
	"go-crm/internal/features/module"
	"go-crm/internal/features/record"
	"slices"
	"strings"
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
	Repo           ApprovalRepository
	RecordRepo     record.RecordRepository
	ModuleRepo     module.ModuleRepository
	UserRepo       user.UserRepository
	AuditService   audit.AuditService
	RoleService    role.RoleService
	ActionExecutor action.ActionExecutor
}

func NewApprovalService(
	repo ApprovalRepository,
	recordRepo record.RecordRepository,
	moduleRepo module.ModuleRepository,
	userRepo user.UserRepository,
	auditService audit.AuditService,
	roleService role.RoleService,
	actionExecutor action.ActionExecutor,
) *ApprovalServiceImpl {
	return &ApprovalServiceImpl{
		Repo:           repo,
		RecordRepo:     recordRepo,
		ModuleRepo:     moduleRepo,
		UserRepo:       userRepo,
		AuditService:   auditService,
		RoleService:    roleService,
		ActionExecutor: actionExecutor,
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
		if s.evaluateCriteria(wf.Criteria, rec) {
			matchedProcess = &wf
			break
		}
	}

	if matchedProcess == nil {
		return nil, nil
	}

	// Find the start node
	startStepID := s.getStartStep(matchedProcess)
	// Resolution for start node if it points to an action
	resolvedStartNode := s.traverseNextStep(matchedProcess, startStepID)

	return &common_models.ApprovalRecordState{
		Status:         common_models.ApprovalStatusPending,
		CurrentStepID:  resolvedStartNode,
		ProcessID:      matchedProcess.ID.Hex(),
		History:        []common_models.ApprovalHistory{},
		CompletedSteps: []string{},
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

	// Backward compatibility: migrate old records using CurrentStep to CurrentStepID
	if state.CurrentStepID == "" && state.CurrentStep >= 0 && state.CurrentStep < len(process.Steps) {
		state.CurrentStepID = process.Steps[state.CurrentStep].ID
		if state.CompletedSteps == nil {
			state.CompletedSteps = []string{}
		}
	}

	// Find current step by ID
	var currentStep *ApprovalStep
	for i := range process.Steps {
		if process.Steps[i].ID == state.CurrentStepID {
			currentStep = &process.Steps[i]
			break
		}
	}

	if currentStep == nil {
		return errors.New("invalid approval step")
	}

	// Add to history
	history := common_models.ApprovalHistory{
		StepName:  currentStep.Name,
		ActorID:   actorID,
		Action:    common_models.ApprovalStatusApproved,
		Comment:   comment,
		Timestamp: time.Now(),
	}
	state.History = append(state.History, history)
	state.CompletedSteps = append(state.CompletedSteps, state.CurrentStepID)

	// Get next steps using graph traversal
	nextSteps := s.getNextSteps(process, state.CurrentStepID)

	if len(nextSteps) == 0 {
		// Terminal step - process complete
		state.Status = common_models.ApprovalStatusApproved
	} else if len(nextSteps) == 1 {
		// Single next step - move to it (resolving any action nodes)
		state.CurrentStepID = s.traverseNextStep(process, nextSteps[0])

		// If resolution returns empty, it means we hit a dead end after actions -> Approved
		if state.CurrentStepID == "" {
			state.Status = common_models.ApprovalStatusApproved
		}
	} else {
		// Multiple next steps - parallel approval
		// For now, take the first one
		// TODO: Implement proper parallel approval handling
		state.CurrentStepID = s.traverseNextStep(process, nextSteps[0])

		if state.CurrentStepID == "" {
			state.Status = common_models.ApprovalStatusApproved
		}
	}

	// Execute After Actions for the current step
	if len(currentStep.AfterActions) > 0 {
		// Use detached context for async actions if preferred, or same context for sync
		// Using sync for now to ensure consistency, or maybe async?
		// Use detached context to ensure actions complete even if request finishes
		detachedCtx := context.WithoutCancel(ctx)
		go s.executeApprovalActions(detachedCtx, currentStep.AfterActions, moduleName, rec)
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

	// Backward compatibility: migrate old records using CurrentStep to CurrentStepID
	if state.CurrentStepID == "" && state.CurrentStep >= 0 && state.CurrentStep < len(process.Steps) {
		state.CurrentStepID = process.Steps[state.CurrentStep].ID
		if state.CompletedSteps == nil {
			state.CompletedSteps = []string{}
		}
	}

	// Find current step by ID
	var currentStep *ApprovalStep
	for i := range process.Steps {
		if process.Steps[i].ID == state.CurrentStepID {
			currentStep = &process.Steps[i]
			break
		}
	}

	if currentStep == nil {
		return errors.New("invalid approval step")
	}

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

	// Backward compatibility: migrate old records using CurrentStep to CurrentStepID
	if state.CurrentStepID == "" && state.CurrentStep >= 0 && state.CurrentStep < len(process.Steps) {
		state.CurrentStepID = process.Steps[state.CurrentStep].ID
	}

	// Find current step by ID
	var currentStep *ApprovalStep
	for i := range process.Steps {
		if process.Steps[i].ID == state.CurrentStepID {
			currentStep = &process.Steps[i]
			break
		}
	}

	if currentStep == nil {
		return false, nil
	}

	// Check if user can approve
	if slices.Contains(currentStep.ApproverUsers, userID) {
		return true, nil
	}

	for _, roleID := range userRoleIDs {
		if slices.Contains(currentStep.ApproverRoles, roleID) {
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

// getNextSteps returns the next step IDs based on the graph edges
// If no edges exist, falls back to sequential order for backward compatibility
func (s *ApprovalServiceImpl) getNextSteps(process *ApprovalProcess, currentStepID string) []string {
	if len(process.Edges) == 0 {
		// Fallback to sequential for backward compatibility
		return s.getNextStepsSequential(process, currentStepID)
	}

	var nextSteps []string
	for _, edge := range process.Edges {
		if edge.Source == currentStepID {
			nextSteps = append(nextSteps, edge.Target)
		}
	}
	return nextSteps
}

func (s *ApprovalServiceImpl) executeApprovalActions(ctx context.Context, actions []ApprovalAction, moduleName string, record map[string]any) {
	if s.ActionExecutor == nil {
		fmt.Println("ActionExecutor is nil, skipping actions")
		return
	}

	var coreActions []action.Action
	for _, act := range actions {
		coreActions = append(coreActions, action.Action{
			Type:   action.ActionType(act.Type),
			Config: act.Config,
		})
	}

	// Inject TenantID from record if context missing it?
	// The context passed from controller/service usually has it.

	if err := s.ActionExecutor.ExecuteActions(ctx, coreActions, moduleName, record); err != nil {
		fmt.Printf("Error executing approval actions: %v\n", err)
	}
}

func (s *ApprovalServiceImpl) evaluateCriteria(criteria []RuleCondition, rec map[string]any) bool {
	if len(criteria) == 0 {
		return true
	}

	for _, cond := range criteria {
		val, exists := rec[cond.Field]
		if !exists {
			return false
		}

		strVal := fmt.Sprintf("%v", val)
		strCond := fmt.Sprintf("%v", cond.Value)

		match := false
		switch cond.Operator {
		case "equals", "eq":
			match = strVal == strCond
		case "not_equals", "neq":
			match = strVal != strCond
		case "contains":
			match = strings.Contains(strVal, strCond)
		case "gt":
			match = strVal > strCond
		case "lt":
			match = strVal < strCond
		case "gte":
			match = strVal >= strCond
		case "lte":
			match = strVal <= strCond
		default:
			// Default to false for unknown operators to prevent accidental trigger
			match = false
		}

		if !match {
			return false
		}
	}
	return true
}

// traverseNextStep recursively skips "action::" nodes to find the next valid "step" ID
func (s *ApprovalServiceImpl) traverseNextStep(process *ApprovalProcess, currentID string) string {
	// 1. If it's empty, we reached end
	if currentID == "" {
		return ""
	}

	// 2. If it's a valid step (not an action node), return it
	// Action nodes in the graph start with "action::"
	// Steps are UUIDs or "start"
	if len(currentID) < 8 || currentID[:8] != "action::" {
		return currentID
	}

	// 3. It's an action node, find what it connects to
	// We assume 1 outgoing edge for actions in this linear flow
	for _, edge := range process.Edges {
		if edge.Source == currentID {
			// Recursively traverse
			return s.traverseNextStep(process, edge.Target)
		}
	}

	// 4. Dead end at an action node (should imply approved/end of flow)
	return ""
}

// getNextStepsSequential provides backward compatibility for processes without edges
func (s *ApprovalServiceImpl) getNextStepsSequential(process *ApprovalProcess, currentStepID string) []string {
	// Find current step index
	currentIdx := -1
	for i, step := range process.Steps {
		if step.ID == currentStepID {
			currentIdx = i
			break
		}
	}

	// Return next step if exists
	if currentIdx >= 0 && currentIdx < len(process.Steps)-1 {
		return []string{process.Steps[currentIdx+1].ID}
	}

	return []string{} // No next steps = process complete
}

// getStartStep finds the first step to execute
// This is the step that has an incoming edge from "start" node
func (s *ApprovalServiceImpl) getStartStep(process *ApprovalProcess) string {
	if len(process.Edges) == 0 {
		// Sequential fallback: first step
		if len(process.Steps) > 0 {
			return process.Steps[0].ID
		}
		return ""
	}

	// Find step connected from "start"
	for _, edge := range process.Edges {
		if edge.Source == "start" {
			return edge.Target
		}
	}

	// Fallback: first step
	if len(process.Steps) > 0 {
		return process.Steps[0].ID
	}
	return ""
}

// isTerminalStep checks if a step has no outgoing edges (terminal node)
func (s *ApprovalServiceImpl) isTerminalStep(process *ApprovalProcess, stepID string) bool {
	if len(process.Edges) == 0 {
		// Sequential mode: last step is terminal
		if len(process.Steps) > 0 {
			return stepID == process.Steps[len(process.Steps)-1].ID
		}
		return true
	}

	// Graph mode: no outgoing edges = terminal
	for _, edge := range process.Edges {
		if edge.Source == stepID {
			return false
		}
	}
	return true
}
