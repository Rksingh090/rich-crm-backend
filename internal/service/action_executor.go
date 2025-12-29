package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"log"
	"net/http"
	"time"

	"github.com/d5/tengo/v2"
)

// ActionExecutor provides centralized action execution for all automation features
type ActionExecutor interface {
	ExecuteActions(ctx context.Context, actions []models.RuleAction, moduleName string, record map[string]interface{}) error
	ExecuteAction(ctx context.Context, action models.RuleAction, moduleName string, record map[string]interface{}) error
}

type ActionExecutorImpl struct {
	recordRepo           repository.RecordRepository
	emailService         EmailService
	emailTemplateService EmailTemplateService
	auditService         AuditService
	syncService          SyncService
	httpClient           *http.Client
}

func NewActionExecutor(
	recordRepo repository.RecordRepository,
	emailService EmailService,
	emailTemplateService EmailTemplateService,
	auditService AuditService,
	syncService SyncService,
) ActionExecutor {
	return &ActionExecutorImpl{
		recordRepo:           recordRepo,
		emailService:         emailService,
		emailTemplateService: emailTemplateService,
		auditService:         auditService,
		syncService:          syncService,
		httpClient:           &http.Client{Timeout: 30 * time.Second},
	}
}

// ExecuteActions executes multiple actions in sequence
func (e *ActionExecutorImpl) ExecuteActions(ctx context.Context, actions []models.RuleAction, moduleName string, record map[string]interface{}) error {
	for i, action := range actions {
		if err := e.ExecuteAction(ctx, action, moduleName, record); err != nil {
			log.Printf("Failed to execute action %d (type: %s): %v", i, action.Type, err)
			// Continue with other actions even if one fails
		}
	}
	return nil
}

// ExecuteAction executes a single action based on its type
func (e *ActionExecutorImpl) ExecuteAction(ctx context.Context, action models.RuleAction, moduleName string, record map[string]interface{}) error {
	switch action.Type {
	case models.ActionSendEmail:
		return e.executeSendEmail(ctx, action.Config, record)

	case models.ActionUpdateField:
		return e.executeUpdateField(ctx, action.Config, moduleName, record)

	case models.ActionWebhook:
		return e.executeWebhook(ctx, action.Config, moduleName, record)

	case models.ActionCreateTask:
		return e.executeCreateTask(ctx, action.Config, record)

	case models.ActionRunScript:
		return e.executeRunScript(ctx, action.Config, moduleName, record)

	case models.ActionSendNotification:
		return e.executeSendNotification(ctx, action.Config, record)

	case models.ActionSendSMS:
		return e.executeSendSMS(ctx, action.Config, record)

	case models.ActionGeneratePDF:
		return e.executeGeneratePDF(ctx, action.Config, moduleName, record)

	case models.ActionDataSync:
		return e.executeDataSync(ctx, action.Config)

	default:
		return fmt.Errorf("unsupported action type: %s", action.Type)
	}
}

// executeSendEmail sends an email based on the action configuration
func (e *ActionExecutorImpl) executeSendEmail(ctx context.Context, config map[string]interface{}, record map[string]interface{}) error {
	to, _ := config["to"].(string)
	subject, _ := config["subject"].(string)
	body, _ := config["body"].(string)

	templateID, _ := config["template_id"].(string)

	if templateID != "" {
		// Use template
		renderedSubject, renderedBody, err := e.emailTemplateService.RenderTemplate(ctx, templateID, record)
		if err != nil {
			return fmt.Errorf("failed to render email template: %w", err)
		}
		subject = renderedSubject
		body = renderedBody
	} else {
		// Replace placeholders in subject and body with record values
		subject = e.replacePlaceholders(subject, record)
		body = e.replacePlaceholders(body, record)
	}

	if to == "" {
		return fmt.Errorf("email recipient (to) is required")
	}

	// Send the email
	log.Printf("Sending email to: %s, subject: %s", to, subject)
	if err := e.emailService.SendEmail(ctx, []string{to}, subject, body); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// executeUpdateField updates a field value in the record
func (e *ActionExecutorImpl) executeUpdateField(ctx context.Context, config map[string]interface{}, moduleName string, record map[string]interface{}) error {
	field, _ := config["field"].(string)
	value := config["value"]

	if field == "" {
		return fmt.Errorf("field name is required for update_field action")
	}

	// Get record ID
	recordID, ok := record["_id"]
	if !ok {
		return fmt.Errorf("record ID not found")
	}

	recordIDStr := fmt.Sprintf("%v", recordID)

	// Update the field
	updateData := map[string]interface{}{
		field: value,
	}

	if err := e.recordRepo.Update(ctx, moduleName, recordIDStr, updateData); err != nil {
		return fmt.Errorf("failed to update field: %w", err)
	}

	log.Printf("Updated field %s to %v for record %s in module %s", field, value, recordIDStr, moduleName)
	return nil
}

// executeWebhook triggers an HTTP webhook
func (e *ActionExecutorImpl) executeWebhook(_ context.Context, config map[string]interface{}, moduleName string, record map[string]interface{}) error {
	url, _ := config["url"].(string)
	method, _ := config["method"].(string)

	if url == "" {
		return fmt.Errorf("webhook URL is required")
	}

	if method == "" {
		method = "POST"
	}

	// Prepare payload
	payload := map[string]interface{}{
		"module":    moduleName,
		"record":    record,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add custom headers if provided
	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			req.Header.Set(key, fmt.Sprintf("%v", value))
		}
	}

	// Send request
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

// executeCreateTask creates a new task record
func (e *ActionExecutorImpl) executeCreateTask(ctx context.Context, config map[string]interface{}, record map[string]interface{}) error {
	subject, _ := config["subject"].(string)
	description, _ := config["description"].(string)
	assignedTo, _ := config["assigned_to"].(string)
	dueDate, _ := config["due_date"].(string)

	if subject == "" {
		return fmt.Errorf("task subject is required")
	}

	// Replace placeholders
	subject = e.replacePlaceholders(subject, record)
	description = e.replacePlaceholders(description, record)

	// Create task data
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

	// Create task in the "tasks" module
	_, err := e.recordRepo.Create(ctx, "tasks", taskData)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	log.Printf("Created task: %s", subject)
	return nil
}

// executeRunScript executes a Tengo script
func (e *ActionExecutorImpl) executeRunScript(_ context.Context, config map[string]interface{}, moduleName string, record map[string]interface{}) error {
	scriptContent, _ := config["script"].(string)

	if scriptContent == "" {
		return fmt.Errorf("script content is required")
	}

	// Compile the script
	script := tengo.NewScript([]byte(scriptContent))

	// Set variables accessible to the script
	script.Add("module", moduleName)
	script.Add("record", record)

	// Run the script
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

// executeSendNotification sends an in-app notification
func (e *ActionExecutorImpl) executeSendNotification(ctx context.Context, config map[string]interface{}, record map[string]interface{}) error {
	userID, _ := config["user_id"].(string)
	title, _ := config["title"].(string)
	message, _ := config["message"].(string)

	if userID == "" {
		return fmt.Errorf("user_id is required for notification")
	}

	if title == "" {
		return fmt.Errorf("notification title is required")
	}

	// Replace placeholders
	title = e.replacePlaceholders(title, record)
	message = e.replacePlaceholders(message, record)

	// Create notification data
	notificationData := map[string]interface{}{
		"user_id":    userID,
		"title":      title,
		"message":    message,
		"read":       false,
		"created_at": time.Now(),
	}

	// Create notification in the "notifications" module
	_, err := e.recordRepo.Create(ctx, "notifications", notificationData)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	log.Printf("Created notification for user %s: %s", userID, title)
	return nil
}

// executeSendSMS sends an SMS message
func (e *ActionExecutorImpl) executeSendSMS(_ context.Context, config map[string]interface{}, record map[string]interface{}) error {
	phoneNumber, _ := config["phone_number"].(string)
	message, _ := config["message"].(string)

	if phoneNumber == "" {
		return fmt.Errorf("phone_number is required for SMS")
	}

	if message == "" {
		return fmt.Errorf("SMS message is required")
	}

	// Replace placeholders
	message = e.replacePlaceholders(message, record)

	// TODO: Integrate with SMS provider (Twilio, AWS SNS, etc.)
	log.Printf("Sending SMS to %s: %s", phoneNumber, message)

	return nil
}

// executeGeneratePDF generates a PDF document
func (e *ActionExecutorImpl) executeGeneratePDF(ctx context.Context, config map[string]interface{}, moduleName string, record map[string]interface{}) error {
	template, _ := config["template"].(string)
	filename, _ := config["filename"].(string)

	if template == "" {
		return fmt.Errorf("PDF template is required")
	}

	if filename == "" {
		filename = fmt.Sprintf("%s_%v.pdf", moduleName, time.Now().Unix())
	}

	// Replace placeholders in template
	template = e.replacePlaceholders(template, record)
	filename = e.replacePlaceholders(filename, record)

	// TODO: Integrate with PDF generation library (e.g., wkhtmltopdf, gotenberg)
	log.Printf("Generating PDF: %s for module %s", filename, moduleName)

	return nil
}

// replacePlaceholders replaces {{field_name}} placeholders with record values
func (e *ActionExecutorImpl) replacePlaceholders(text string, record map[string]interface{}) string {
	// Simple placeholder replacement
	// TODO: Implement more sophisticated template engine if needed
	for key, value := range record {
		placeholder := fmt.Sprintf("{{%s}}", key)
		replacement := fmt.Sprintf("%v", value)
		text = replaceAll(text, placeholder, replacement)
	}
	return text
}

// Helper function to replace all occurrences
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

// Helper function to find index of substring
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// executeDataSync triggers a data synchronization
func (e *ActionExecutorImpl) executeDataSync(ctx context.Context, config map[string]interface{}) error {
	syncSettingID, _ := config["sync_setting_id"].(string)

	if syncSettingID == "" {
		return fmt.Errorf("sync_setting_id is required for data_sync action")
	}

	log.Printf("Triggering data sync for setting ID: %s", syncSettingID)

	// Execute the sync
	if err := e.syncService.RunSync(ctx, syncSettingID); err != nil {
		return fmt.Errorf("data sync failed: %w", err)
	}

	log.Printf("Data sync completed successfully for setting ID: %s", syncSettingID)
	return nil
}
