package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"strings"

	"net/http"
	"time"

	"github.com/d5/tengo/v2"
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
	Repo         repository.AutomationRepository
	RecordRepo   repository.RecordRepository
	AuditService AuditService
	EmailService EmailService
	HttpClient   *http.Client
}

func NewAutomationService(repo repository.AutomationRepository, recordRepo repository.RecordRepository, auditService AuditService, emailService EmailService) AutomationService {
	return &AutomationServiceImpl{
		Repo:         repo,
		RecordRepo:   recordRepo,
		AuditService: auditService,
		EmailService: emailService,
		HttpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *AutomationServiceImpl) CreateRule(ctx context.Context, rule *models.AutomationRule) error {
	return s.Repo.Create(ctx, rule)
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
	return s.Repo.Update(ctx, rule)
}

func (s *AutomationServiceImpl) DeleteRule(ctx context.Context, id string) error {
	return s.Repo.Delete(ctx, id)
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
	for _, action := range actions {
		switch action.Type {
		case models.ActionCreateTask:
			// Config: { "subject": "Task Name", "due_in_hours": 24 }
			// Logic: Create record in "tasks" module
			// This requires "tasks" module to exist and "RecordRepo" usage
			// For MVP: Just log or implement generic record creation if easy
			fmt.Printf("Automation Action: Create Task %v\n", action.Config)

		case models.ActionUpdateField:
			// Config: { "field": "status", "value": "Contacted" }
			fieldName, _ := action.Config["field"].(string)
			newVal := action.Config["value"]
			if fieldName != "" {
				if idRaw, ok := record["_id"]; ok {
					// Need to convert objectID to string hex
					idHex := fmt.Sprintf("%v", idRaw)
					// Use RecordRepo directly? Or Service?
					// RecordService calls AutomationService, so calling RecordService here causes Cycle.
					// Must use RecordRepo.

					// Dangerous: We are bypassing Service validation (schema, readonly, etc.)
					// But usually automation has "sudo" rights.
					err := s.RecordRepo.Update(ctx, moduleName, strings.TrimPrefix(strings.TrimSuffix(idHex, ")"), "ObjectID(\""), map[string]interface{}{fieldName: newVal})
					if err != nil {
						return err
					}
					// fmt.Printf("Automation Action: Updated field %s to %v\n", fieldName, newVal)

					// Log to Audit
					if s.AuditService != nil {
						changes := map[string]models.Change{
							fieldName: {New: newVal},
						}
						// RecordID? We have it in `record["_id"]`, but need to parse hex
						recID := strings.TrimPrefix(strings.TrimSuffix(idHex, ")"), "ObjectID(\"")
						_ = s.AuditService.LogChange(ctx, models.AuditActionAutomation, moduleName, recID, changes)
					}
				}
			}

		case models.ActionSendEmail:
			// Config: { "to": "...", "subject": "...", "body": "..." }
			fmt.Printf("Automation Action: Send Email %v\n", action.Config)

			if s.EmailService != nil {
				// Parse recipients. Config 'to' might be string (comma user) or array
				toStr, ok := action.Config["to"].(string)
				if ok && toStr != "" {
					to := []string{toStr} // TODO: Handle multiple/comma separated
					subject, _ := action.Config["subject"].(string)
					body, _ := action.Config["body"].(string)

					// Send in background? Or sync?
					// Sync ensures we know if it failed, but might block.
					// Let's assume sync for now.
					if err := s.EmailService.SendEmail(ctx, to, subject, body); err != nil {
						fmt.Printf("Failed to send email: %v\n", err)
						// Don't return error to stop other actions? Or return?
						// Just log for now.
					}
				}
			}

			// Log to Audit
			if s.AuditService != nil {
				// Try to get Record ID
				recID := ""
				if idRaw, ok := record["_id"]; ok {
					idHex := fmt.Sprintf("%v", idRaw)
					recID = strings.TrimPrefix(strings.TrimSuffix(idHex, ")"), "ObjectID(\"")
				}
				_ = s.AuditService.LogChange(ctx, models.AuditActionAutomation, moduleName, recID, map[string]models.Change{
					"action": {New: fmt.Sprintf("Sent Email: %v", action.Config["subject"])},
				})
			}

		case models.ActionWebhook:
			// Config: { "url": "...", "method": "POST", "headers": {...} }
			url, _ := action.Config["url"].(string)
			if url == "" {
				continue
			}

			method := "POST"
			if m, ok := action.Config["method"].(string); ok && m != "" {
				method = m
			}

			payload := map[string]interface{}{
				"event":     "automation.trigger",
				"module":    moduleName,
				"data":      record,
				"timestamp": time.Now(),
			}

			body, _ := json.Marshal(payload)
			req, err := http.NewRequest(method, url, strings.NewReader(string(body)))
			if err != nil {
				fmt.Printf("Failed to create webhook request: %v\n", err)
				continue
			}

			req.Header.Set("Content-Type", "application/json")
			if headers, ok := action.Config["headers"].(map[string]interface{}); ok {
				for k, v := range headers {
					req.Header.Set(k, fmt.Sprintf("%v", v))
				}
			}

			resp, err := s.HttpClient.Do(req)
			if err != nil {
				fmt.Printf("Failed to trigger webhook: %v\n", err)
				continue
			}
			resp.Body.Close()

			// Log to Audit
			if s.AuditService != nil {
				recID := ""
				if idRaw, ok := record["_id"]; ok {
					idHex := fmt.Sprintf("%v", idRaw)
					recID = strings.TrimPrefix(strings.TrimSuffix(idHex, ")"), "ObjectID(\"")
				}
				_ = s.AuditService.LogChange(ctx, models.AuditActionAutomation, moduleName, recID, map[string]models.Change{
					"action": {New: fmt.Sprintf("Triggered Webhook: %s", url)},
				})
			}

		case models.ActionRunScript:
			scriptContent, _ := action.Config["script_content"].(string)
			// Fallback for backward compatibility or if user used the old UI
			if scriptContent == "" {
				scriptContent, _ = action.Config["script"].(string)
				// If it was a name (e.g. "deduct_inventory"), we strictly need content now.
				// For this transition, we might fails if it's just a name
				// unless we load it from DB?
				// Let's assume the UI sends content now.
			}

			if scriptContent != "" {
				if err := s.executeDynamicScript(ctx, scriptContent, moduleName, record); err != nil {
					fmt.Printf("Error running dynamic script: %v\n", err)
				} else {
					if s.AuditService != nil {
						recID := ""
						if idRaw, ok := record["_id"]; ok {
							idHex := fmt.Sprintf("%v", idRaw)
							recID = strings.TrimPrefix(strings.TrimSuffix(idHex, ")"), "ObjectID(\"")
						}
						_ = s.AuditService.LogChange(ctx, models.AuditActionAutomation, moduleName, recID, map[string]models.Change{
							"action": {New: "Ran Dynamic Script"},
						})
					}
				}
			}
		}
	}
	return nil
}

func (s *AutomationServiceImpl) executeDynamicScript(ctx context.Context, scriptContent string, moduleName string, record map[string]interface{}) error {
	script := tengo.NewScript([]byte(scriptContent))

	// Define 'modules' object
	modulesMap := map[string]tengo.Object{
		"get": &tengo.UserFunction{
			Name: "get",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 2 {
					return nil, tengo.ErrWrongNumArguments
				}
				mod, ok1 := tengo.ToString(args[0])
				id, ok2 := tengo.ToString(args[1])
				if !ok1 || !ok2 {
					return nil, tengo.ErrInvalidArgumentType{
						Name: "first and second arguments must be string",
					}
				}

				res, err := s.RecordRepo.Get(ctx, mod, id)
				if err != nil {
					return tengo.UndefinedValue, nil // Return undefined on not found/error
				}
				return tengo.FromInterface(res)
			},
		},
		"list": &tengo.UserFunction{
			Name: "list",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 2 {
					return nil, tengo.ErrWrongNumArguments
				}
				mod, ok1 := tengo.ToString(args[0])
				filterMap, ok2 := tengo.ToInterface(args[1]).(map[string]interface{})
				if !ok1 || !ok2 {
					// try converting filter directly?
					// Tengo objects to interface map
					// simpler: return error
					return nil, tengo.ErrInvalidArgumentType{
						Name: "arguments: module(string), filter(map)",
					}
				}

				// Convert filter values
				f := make(map[string]interface{})
				for k, v := range filterMap {
					f[k] = v
				}

				res, err := s.RecordRepo.List(ctx, mod, f, 100, 0, "created_at", -1)
				if err != nil {
					return &tengo.Array{}, nil
				}
				return tengo.FromInterface(res)
			},
		},
		"update": &tengo.UserFunction{
			Name: "update",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 3 {
					return nil, tengo.ErrWrongNumArguments
				}
				mod, ok1 := tengo.ToString(args[0])
				id, ok2 := tengo.ToString(args[1])
				dataMap, ok3 := tengo.ToInterface(args[2]).(map[string]interface{})

				if !ok1 || !ok2 || !ok3 {
					return nil, tengo.ErrInvalidArgumentType{
						Name: "module(string), id(string), data(map)",
					}
				}

				// Only allow updating non-system fields?
				// For now allow all except _id
				delete(dataMap, "_id")
				delete(dataMap, "id")

				err := s.RecordRepo.Update(ctx, mod, id, dataMap)
				if err != nil {
					return tengo.FalseValue, nil
				}
				return tengo.TrueValue, nil
			},
		},
	}

	script.Add("modules", &tengo.ImmutableMap{Value: modulesMap})

	// Add 'record_id' and 'module_name' as context
	if idRaw, ok := record["_id"]; ok {
		idHex := fmt.Sprintf("%v", idRaw)
		recID := strings.TrimPrefix(strings.TrimSuffix(idHex, ")"), "ObjectID(\"")
		script.Add("record_id", recID)
	}
	script.Add("context_module", moduleName)

	// Add simple logging
	script.Add("log", &tengo.UserFunction{
		Name: "log",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			for _, arg := range args {
				fmt.Printf("[SCRIPT LOG] %v ", arg)
			}
			fmt.Println()
			return tengo.UndefinedValue, nil
		},
	})

	_, err := script.Run()
	return err
}
