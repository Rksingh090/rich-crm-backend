package automation

import (
	"context"
	"encoding/json"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/email"
	"go-crm/internal/features/email_template"
	"go-crm/internal/features/module"
	"go-crm/internal/features/record"
	"go-crm/internal/features/sync"
	"log"
	"net/http"
	"time"

	"github.com/d5/tengo/v2"
)

// ActionExecutor provides centralized action execution for all automation features
type ActionExecutor interface {
	ExecuteActions(ctx context.Context, actions []RuleAction, moduleName string, record map[string]interface{}) error
	ExecuteAction(ctx context.Context, action RuleAction, moduleName string, record map[string]interface{}) error
}

type ActionExecutorImpl struct {
	moduleRepo           module.ModuleRepository
	recordRepo           record.RecordRepository
	emailService         email.EmailService
	emailTemplateService email_template.EmailTemplateService
	auditService         audit.AuditService
	syncService          sync.SyncService
	httpClient           *http.Client
}

func NewActionExecutor(
	moduleRepo module.ModuleRepository,
	recordRepo record.RecordRepository,
	emailService email.EmailService,
	emailTemplateService email_template.EmailTemplateService,
	auditService audit.AuditService,
	syncService sync.SyncService,
) ActionExecutor {
	return &ActionExecutorImpl{
		moduleRepo:           moduleRepo,
		recordRepo:           recordRepo,
		emailService:         emailService,
		emailTemplateService: emailTemplateService,
		auditService:         auditService,
		syncService:          syncService,
		httpClient:           &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *ActionExecutorImpl) ExecuteActions(ctx context.Context, actions []RuleAction, moduleName string, record map[string]interface{}) error {
	for i, action := range actions {
		if err := e.ExecuteAction(ctx, action, moduleName, record); err != nil {
			log.Printf("Failed to execute action %d (type: %s): %v", i, action.Type, err)
		}
	}
	return nil
}

func (e *ActionExecutorImpl) ExecuteAction(ctx context.Context, action RuleAction, moduleName string, record map[string]interface{}) error {
	switch action.Type {
	case ActionSendEmail:
		return e.executeSendEmail(ctx, action.Config, record)

	case ActionUpdateField:
		return e.executeUpdateField(ctx, action.Config, moduleName, record)

	case ActionWebhook:
		return e.executeWebhook(ctx, action.Config, moduleName, record)

	case ActionCreateTask:
		return e.executeCreateTask(ctx, action.Config, record)

	case ActionRunScript:
		return e.executeRunScript(ctx, action.Config, moduleName, record)

	case ActionSendNotification:
		return e.executeSendNotification(ctx, action.Config, record)

	case ActionSendSMS:
		return e.executeSendSMS(ctx, action.Config, record)

	case ActionGeneratePDF:
		return e.executeGeneratePDF(ctx, action.Config, moduleName, record)

	case ActionDataSync:
		return e.executeDataSync(ctx, action.Config)

	default:
		return fmt.Errorf("unsupported action type: %s", action.Type)
	}
}

func (e *ActionExecutorImpl) executeSendEmail(ctx context.Context, config map[string]interface{}, rec map[string]interface{}) error {
	to, _ := config["to"].(string)
	subject, _ := config["subject"].(string)
	body, _ := config["body"].(string)

	templateID, _ := config["template_id"].(string)

	if templateID != "" {
		renderedSubject, renderedBody, err := e.emailTemplateService.RenderTemplate(ctx, templateID, rec)
		if err != nil {
			return fmt.Errorf("failed to render email template: %w", err)
		}
		subject = renderedSubject
		body = renderedBody
	} else {
		subject = e.replacePlaceholders(subject, rec)
		body = e.replacePlaceholders(body, rec)
	}

	if to == "" {
		return fmt.Errorf("email recipient (to) is required")
	}

	log.Printf("Sending email to: %s, subject: %s", to, subject)
	if err := e.emailService.SendEmail(ctx, []string{to}, subject, body); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (e *ActionExecutorImpl) executeUpdateField(ctx context.Context, config map[string]interface{}, moduleName string, rec map[string]interface{}) error {
	field, _ := config["field"].(string)
	value := config["value"]

	if field == "" {
		return fmt.Errorf("field name is required for update_field action")
	}

	recordID, ok := rec["_id"]
	if !ok {
		return fmt.Errorf("record ID not found")
	}

	recordIDStr := fmt.Sprintf("%v", recordID)

	updateData := map[string]interface{}{
		field: value,
	}

	if err := e.recordRepo.Update(ctx, moduleName, recordIDStr, updateData); err != nil {
		return fmt.Errorf("failed to update field: %w", err)
	}

	log.Printf("Updated field %s to %v for record %s in module %s", field, value, recordIDStr, moduleName)
	return nil
}

func (e *ActionExecutorImpl) executeWebhook(_ context.Context, config map[string]interface{}, moduleName string, rec map[string]interface{}) error {
	url, _ := config["url"].(string)
	method, _ := config["method"].(string)

	if url == "" {
		return fmt.Errorf("webhook URL is required")
	}

	if method == "" {
		method = "POST"
	}

	payload := map[string]interface{}{
		"module":    moduleName,
		"record":    rec,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			req.Header.Set(key, fmt.Sprintf("%v", value))
		}
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("Webhook sent to %s, status: %d, payload: %s", url, resp.StatusCode, string(payloadBytes))

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}

	return nil
}

func (e *ActionExecutorImpl) executeCreateTask(ctx context.Context, config map[string]interface{}, rec map[string]interface{}) error {
	subject, _ := config["subject"].(string)
	description, _ := config["description"].(string)
	assignedTo, _ := config["assigned_to"].(string)
	dueDate, _ := config["due_date"].(string)

	if subject == "" {
		return fmt.Errorf("task subject is required")
	}

	subject = e.replacePlaceholders(subject, rec)
	description = e.replacePlaceholders(description, rec)

	taskData := map[string]interface{}{
		"subject":     subject,
		"description": description,
		"status":      "pending",
		"created_at":  time.Now(),
	}

	if assignedTo != "" {
		taskData["assigned_to"] = assignedTo
	}

	if dueDate != "" {
		taskData["due_date"] = dueDate
	}

	// Lookup tasks module to get product
	taskModule, err := e.moduleRepo.FindByName(ctx, "tasks")
	var product common_models.Product = common_models.ProductCRM // Default
	if err == nil {
		product = taskModule.Product
	}

	_, err = e.recordRepo.Create(ctx, "tasks", product, taskData)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	log.Printf("Created task: %s", subject)
	return nil
}

func (e *ActionExecutorImpl) executeRunScript(_ context.Context, config map[string]interface{}, moduleName string, rec map[string]interface{}) error {
	scriptContent, _ := config["script"].(string)

	if scriptContent == "" {
		return fmt.Errorf("script content is required")
	}

	script := tengo.NewScript([]byte(scriptContent))

	script.Add("module", moduleName)
	script.Add("record", rec)

	compiled, err := script.Compile()
	if err != nil {
		return fmt.Errorf("failed to compile script: %w", err)
	}

	if err := compiled.Run(); err != nil {
		return fmt.Errorf("failed to run script: %w", err)
	}

	log.Printf("Executed script for module %s", moduleName)
	return nil
}

func (e *ActionExecutorImpl) executeSendNotification(ctx context.Context, config map[string]interface{}, rec map[string]interface{}) error {
	userID, _ := config["user_id"].(string)
	title, _ := config["title"].(string)
	message, _ := config["message"].(string)

	if userID == "" {
		return fmt.Errorf("user_id is required for notification")
	}

	if title == "" {
		return fmt.Errorf("notification title is required")
	}

	title = e.replacePlaceholders(title, rec)
	message = e.replacePlaceholders(message, rec)

	notificationData := map[string]interface{}{
		"user_id":    userID,
		"title":      title,
		"message":    message,
		"read":       false,
		"created_at": time.Now(),
	}

	// Lookup notifications module to get product
	notifModule, err := e.moduleRepo.FindByName(ctx, "notifications")
	var product common_models.Product = common_models.ProductCRM // Default or Platform?
	if err == nil {
		product = notifModule.Product
	}

	_, err = e.recordRepo.Create(ctx, "notifications", product, notificationData)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	log.Printf("Created notification for user %s: %s", userID, title)
	return nil
}

func (e *ActionExecutorImpl) executeSendSMS(_ context.Context, config map[string]interface{}, rec map[string]interface{}) error {
	phoneNumber, _ := config["phone_number"].(string)
	message, _ := config["message"].(string)

	if phoneNumber == "" {
		return fmt.Errorf("phone_number is required for SMS")
	}

	if message == "" {
		return fmt.Errorf("SMS message is required")
	}

	message = e.replacePlaceholders(message, rec)

	log.Printf("Sending SMS to %s: %s", phoneNumber, message)

	return nil
}

func (e *ActionExecutorImpl) executeGeneratePDF(ctx context.Context, config map[string]interface{}, moduleName string, rec map[string]interface{}) error {
	template, _ := config["template"].(string)
	filename, _ := config["filename"].(string)

	if template == "" {
		return fmt.Errorf("PDF template is required")
	}

	if filename == "" {
		filename = fmt.Sprintf("%s_%v.pdf", moduleName, time.Now().Unix())
	}

	template = e.replacePlaceholders(template, rec)
	filename = e.replacePlaceholders(filename, rec)

	log.Printf("Generating PDF: %s for module %s", filename, moduleName)

	return nil
}

func (e *ActionExecutorImpl) replacePlaceholders(text string, rec map[string]interface{}) string {
	for key, value := range rec {
		placeholder := fmt.Sprintf("{{%s}}", key)
		replacement := fmt.Sprintf("%v", value)
		text = replaceAll(text, placeholder, replacement)
	}
	return text
}

func replaceAll(s, old, new string) string {
	result := ""
	for {
		i := indexOf(s, old)
		if i == -1 {
			result += s
			break
		}
		result += s[:i] + new
		s = s[i+len(old):]
	}
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func (e *ActionExecutorImpl) executeDataSync(ctx context.Context, config map[string]interface{}) error {
	syncSettingID, _ := config["sync_setting_id"].(string)

	if syncSettingID == "" {
		return fmt.Errorf("sync_setting_id is required for data_sync action")
	}

	log.Printf("Triggering data sync for setting ID: %s", syncSettingID)

	if err := e.syncService.RunSync(ctx, syncSettingID); err != nil {
		return fmt.Errorf("data sync failed: %w", err)
	}

	log.Printf("Data sync completed successfully for setting ID: %s", syncSettingID)
	return nil
}
