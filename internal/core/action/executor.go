package action

import (
	"context"
	"encoding/json"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/core/audit"
	"go-crm/internal/features/email"
	"go-crm/internal/features/email_template"
	"go-crm/internal/features/module"
	"go-crm/internal/features/record"
	"go-crm/internal/features/sync"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/d5/tengo/v2"
)

// ActionExecutor provides centralized action execution for all automation features
type ActionExecutor interface {
	ExecuteActions(ctx context.Context, actions []Action, moduleName string, record map[string]interface{}) error
	ExecuteAction(ctx context.Context, action Action, moduleName string, record map[string]interface{}) error
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

func (e *ActionExecutorImpl) ExecuteActions(ctx context.Context, actions []Action, moduleName string, record map[string]interface{}) error {
	for i, action := range actions {
		if err := e.ExecuteAction(ctx, action, moduleName, record); err != nil {
			log.Printf("Failed to execute action %d (type: %s): %v", i, action.Type, err)
		}
	}
	return nil
}

func (e *ActionExecutorImpl) ExecuteAction(ctx context.Context, action Action, moduleName string, record map[string]interface{}) error {
	switch action.Type {
	case ActionSendEmail:
		return e.executeSendEmail(ctx, action.Config, record)

	case ActionUpdateField:
		return e.executeUpdateField(ctx, action.Config, moduleName, record)

	case ActionWebhook:
		return e.executeWebhook(ctx, action.Config, moduleName, record)

	case ActionCreateRecord:
		return e.executeCreateRecord(ctx, action.Config, record)

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
	cc, _ := config["cc"].(string)
	bcc, _ := config["bcc"].(string)
	from, _ := config["from"].(string)
	subject, _ := config["subject"].(string)
	body, _ := config["body"].(string)
	templateID := ""
	if tid, ok := config["template_id"]; ok {
		templateID = fmt.Sprintf("%v", tid)
	}

	if templateID != "" && templateID != "none" {
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

	to = e.replacePlaceholders(to, rec)
	cc = e.replacePlaceholders(cc, rec)
	bcc = e.replacePlaceholders(bcc, rec)
	from = e.replacePlaceholders(from, rec)

	if to == "" {
		return fmt.Errorf("email recipient (to) is required")
	}

	parseEmails := func(s string) []string {
		if s == "" {
			return nil
		}
		parts := strings.Split(s, ",")
		var result []string
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	opts := email.EmailOptions{
		To:      parseEmails(to),
		Cc:      parseEmails(cc),
		Bcc:     parseEmails(bcc),
		From:    from,
		Subject: subject,
		Body:    body,
	}

	log.Printf("Sending email to: %v, CC: %v, BCC: %v, subject: %s", opts.To, opts.Cc, opts.Bcc, opts.Subject)
	if err := e.emailService.SendEmailWithOptions(ctx, opts); err != nil {
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

func (e *ActionExecutorImpl) executeCreateRecord(ctx context.Context, config map[string]interface{}, rec map[string]interface{}) error {
	targetModuleName, _ := config["module_name"].(string)
	if targetModuleName == "" {
		return fmt.Errorf("target module_name is required for create_record action")
	}

	mapping, _ := config["mapping"].(map[string]interface{})
	if mapping == nil {
		mapping = make(map[string]interface{})
	}

	newData := make(map[string]interface{})
	for targetField, sourceFieldRaw := range mapping {
		sourceField, ok := sourceFieldRaw.(string)
		if !ok || sourceField == "" {
			continue
		}

		// Get value from source record
		if val, ok := rec[sourceField]; ok {
			newData[targetField] = val
		}
	}

	// Always set common fields if they don't exist
	if _, ok := newData["created_at"]; !ok {
		newData["created_at"] = time.Now()
	}

	// Lookup target module to get product
	targetModule, err := e.moduleRepo.FindByName(ctx, targetModuleName)
	var product common_models.App = common_models.AppCRM // Default
	if err == nil {
		product = targetModule.App
	}

	_, err = e.recordRepo.Create(ctx, targetModuleName, product, newData)
	if err != nil {
		return fmt.Errorf("failed to create record in module %s: %w", targetModuleName, err)
	}

	log.Printf("Created record in module %s using mapping", targetModuleName)
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
	var product common_models.App = common_models.AppCRM // Default or Platform?
	if err == nil {
		product = notifModule.App
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
