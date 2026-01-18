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
		isMatch := rule.TriggerType == triggerType
		if rule.TriggerType == "create_or_update" && (triggerType == "create" || triggerType == "update") {
			isMatch = true
		}

		if !isMatch {
			continue
		}

		// Inject TenantID into context
		execCtx := context.WithValue(ctx, common_models.TenantIDKey, rule.TenantID.Hex())

		// If VisualLayout is present, use graph execution
		if rule.VisualLayout != nil && len(rule.VisualLayout.Nodes) > 0 {
			if err := s.executeGraph(execCtx, rule, record, moduleName); err != nil {
				fmt.Printf("Error executing graph rule '%s': %v\n", rule.Name, err)
			}
		} else {
			// Fallback to legacy linear execution
			if s.evaluateConditions(rule.Conditions, record) {
				if err := s.executeActions(execCtx, rule.Actions, moduleName, record); err != nil {
					fmt.Printf("Error executing linear rule '%s': %v\n", rule.Name, err)
				}
			}
		}
	}
	return nil
}

func (s *AutomationServiceImpl) executeGraph(ctx context.Context, rule AutomationRule, record map[string]any, moduleName string) error {
	startNode := s.findStartNode(rule.VisualLayout.Nodes)
	if startNode == nil {
		return fmt.Errorf("no start node found")
	}

	return s.traverse(ctx, startNode, rule.VisualLayout, record, moduleName)
}

func (s *AutomationServiceImpl) findStartNode(nodes []VisualNode) *VisualNode {
	for _, node := range nodes {
		if node.Type == "trigger" {
			return &node
		}
	}
	return nil
}

func (s *AutomationServiceImpl) traverse(ctx context.Context, currentNode *VisualNode, layout *VisualLayout, record map[string]any, moduleName string) error {
	// Find next nodes based on edges
	outgoingEdges := s.findOutgoingEdges(currentNode.ID, layout.Edges)

	// Process current node logic before moving to next
	// Trigger node is just a start point, no logic needed here effectively as we are already triggered.

	// Determine which path to take
	var targetNodeID string

	switch currentNode.Type {
	case "trigger":
		// Triggers usually have one output, just follow it
		if len(outgoingEdges) > 0 {
			targetNodeID = outgoingEdges[0].Target
		}

	case "condition":
		// Evaluate condition
		conditionMet := s.evaluateVisualCondition(currentNode.Data, record)

		// Find edge matching criteria
		for _, edge := range outgoingEdges {
			if conditionMet && edge.SourceHandle == "true" {
				targetNodeID = edge.Target
				break
			}
			if !conditionMet && edge.SourceHandle == "false" {
				targetNodeID = edge.Target
				break
			}
		}

	case "action":
		// Execute action
		if err := s.executeVisualAction(ctx, currentNode.Data, moduleName, record); err != nil {
			return err
		}
		// Actions usually have one output or none (end)
		if len(outgoingEdges) > 0 {
			targetNodeID = outgoingEdges[0].Target
		}
	}

	if targetNodeID != "" {
		nextNode := s.findNodeByID(targetNodeID, layout.Nodes)
		if nextNode != nil {
			return s.traverse(ctx, nextNode, layout, record, moduleName)
		}
	}

	return nil
}

func (s *AutomationServiceImpl) findOutgoingEdges(nodeID string, edges []VisualEdge) []VisualEdge {
	var result []VisualEdge
	for _, edge := range edges {
		if edge.Source == nodeID {
			result = append(result, edge)
		}
	}
	return result
}

func (s *AutomationServiceImpl) findNodeByID(id string, nodes []VisualNode) *VisualNode {
	for _, node := range nodes {
		if node.ID == id {
			return &node
		}
	}
	return nil
}

func (s *AutomationServiceImpl) evaluateVisualCondition(data map[string]interface{}, record map[string]any) bool {
	// Extract condition data
	field, _ := data["field"].(string)
	operator, _ := data["operator"].(string)
	value := data["value"]

	if field == "" {
		return true // Missing config treated as pass or fail? Let's say pass for now or make it robust.
	}

	val, exists := record[field]
	if !exists {
		return false
	}

	// Map generic operator string to enum or just compare
	// Reusing logic from evaluateConditions but adapted for single item
	match := false
	switch operator {
	case "equals":
		match = fmt.Sprintf("%v", val) == fmt.Sprintf("%v", value)
	case "not_equals":
		match = fmt.Sprintf("%v", val) != fmt.Sprintf("%v", value)
	case "contains":
		match = strings.Contains(fmt.Sprintf("%v", val), fmt.Sprintf("%v", value))
	case "gt":
		// Simple string comparison for now, valid for numbers if formatted correctly or need type casting
		match = fmt.Sprintf("%v", val) > fmt.Sprintf("%v", value)
	case "lt":
		match = fmt.Sprintf("%v", val) < fmt.Sprintf("%v", value)
	default:
		match = false
	}

	return match
}

func (s *AutomationServiceImpl) executeVisualAction(ctx context.Context, data map[string]interface{}, moduleName string, record map[string]any) error {
	actionType, _ := data["type"].(string)
	config, _ := data["config"].(map[string]interface{})

	if actionType == "" {
		return nil
	}

	// Construct Action struct
	act := action.Action{
		Type:   action.ActionType(actionType),
		Config: config,
	}

	// Execute single action
	return s.ActionExecutor.ExecuteActions(ctx, []action.Action{act}, moduleName, record)
}

// Keep legacy methods for fallback
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
