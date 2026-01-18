package automation

import (
	"go-crm/internal/core/action"
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

// Trigger Types
type TriggerType string

const (
	TriggerRecordCreated TriggerType = "record_created"
	TriggerRecordUpdated TriggerType = "record_updated"
	TriggerRecordDeleted TriggerType = "record_deleted"
	TriggerFieldChanged  TriggerType = "field_changed"
	TriggerScheduled     TriggerType = "scheduled"
)

type RuleCondition struct {
	Field    string             `json:"field" bson:"field"`
	Operator ValidationOperator `json:"operator" bson:"operator"`
	Value    any                `json:"value" bson:"value"`
}

type AutomationRule struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID `json:"tenant_id" bson:"tenant_id"`
	Name        string             `json:"name" bson:"name"`
	ModuleID    string             `json:"module_id" bson:"module_id"`
	TriggerType string             `json:"trigger_type" bson:"trigger_type"`
	Active      bool               `json:"active" bson:"active"`
	Conditions  []RuleCondition    `json:"conditions" bson:"conditions"`
	Actions     []action.Action    `json:"actions" bson:"actions"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}
