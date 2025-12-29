package models

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
	// Add more as needed
)

type ActionType string

const (
	ActionSendEmail        ActionType = "send_email"
	ActionCreateTask       ActionType = "create_task" // Creates a record in "tasks" module
	ActionUpdateField      ActionType = "update_field"
	ActionWebhook          ActionType = "webhook"           // Custom HTTP trigger
	ActionRunScript        ActionType = "run_script"        // Runs a registered Go script
	ActionSendNotification ActionType = "send_notification" // In-app notification
	ActionSendSMS          ActionType = "send_sms"          // Send SMS message
	ActionGeneratePDF      ActionType = "generate_pdf"      // Generate PDF document
	ActionDataSync         ActionType = "data_sync"         // Trigger data synchronization
)

type RuleCondition struct {
	Field    string             `json:"field" bson:"field"`
	Operator ValidationOperator `json:"operator" bson:"operator"`
	Value    interface{}        `json:"value" bson:"value"`
}

type RuleAction struct {
	Type   ActionType             `json:"type" bson:"type"`
	Config map[string]interface{} `json:"config" bson:"config"` // e.g., { "to": "user@example.com", "subject": "..." } or { "field": "status", "value": "Contacted" }
}

type AutomationRule struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	ModuleID    string             `json:"module_id" bson:"module_id"`       // Target Module Name or ID
	TriggerType string             `json:"trigger_type" bson:"trigger_type"` // "create", "update", "delete"
	Active      bool               `json:"active" bson:"active"`
	Conditions  []RuleCondition    `json:"conditions" bson:"conditions"`
	Actions     []RuleAction       `json:"actions" bson:"actions"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}
