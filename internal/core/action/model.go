package action

// ActionType represents the type of action to execute
type ActionType string

const (
	ActionSendEmail        ActionType = "send_email"
	ActionCreateRecord     ActionType = "create_record"
	ActionWebhook          ActionType = "webhook"
	ActionRunFunction      ActionType = "run_function"
	ActionSendNotification ActionType = "send_notification"
	ActionDataSync         ActionType = "data_sync"
)

// Action represents a single action to be executed
type Action struct {
	Type   ActionType     `json:"type" bson:"type"`
	Config map[string]any `json:"config" bson:"config"`
}
