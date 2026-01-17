package action

// ActionType represents the type of action to execute
type ActionType string

const (
	ActionSendEmail        ActionType = "send_email"
	ActionCreateTask       ActionType = "create_task"
	ActionUpdateField      ActionType = "update_field"
	ActionWebhook          ActionType = "webhook"
	ActionRunScript        ActionType = "run_script"
	ActionSendNotification ActionType = "send_notification"
	ActionSendSMS          ActionType = "send_sms"
	ActionGeneratePDF      ActionType = "generate_pdf"
	ActionDataSync         ActionType = "data_sync"
	ActionSendReport       ActionType = "send_report"
)

// Action represents a single action to be executed
type Action struct {
	Type   ActionType             `json:"type" bson:"type"`
	Config map[string]interface{} `json:"config" bson:"config"`
}
