package automation

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ValidationOperator string

const (
	OperatorEquals      ValidationOperator = "equals"
	OperatorNotEquals   ValidationOperator = "not_equals"
	OperatorContains    ValidationOperator = "contains"
	OperatorGreaterThan ValidationOperator = "gt"
	OperatorLessThan    ValidationOperator = "lt"
)

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

type RuleCondition struct {
	Field    string             `json:"field" bson:"field"`
	Operator ValidationOperator `json:"operator" bson:"operator"`
	Value    interface{}        `json:"value" bson:"value"`
}

type RuleAction struct {
	Type   ActionType             `json:"type" bson:"type"`
	Config map[string]interface{} `json:"config" bson:"config"`
}

type AutomationRule struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID `json:"tenant_id" bson:"tenant_id"`
	Name        string             `json:"name" bson:"name"`
	ModuleID    string             `json:"module_id" bson:"module_id"`
	TriggerType string             `json:"trigger_type" bson:"trigger_type"`
	Active      bool               `json:"active" bson:"active"`
	Conditions  []RuleCondition    `json:"conditions" bson:"conditions"`
	Actions     []RuleAction       `json:"actions" bson:"actions"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}
