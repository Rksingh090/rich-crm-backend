package cron_feature

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

// CronJob represents a scheduled automation job
type CronJob struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Schedule    string             `json:"schedule" bson:"schedule"`
	ModuleID    string             `json:"module_id,omitempty" bson:"module_id,omitempty"`
	Conditions  []RuleCondition    `json:"conditions,omitempty" bson:"conditions,omitempty"`
	Actions     []RuleAction       `json:"actions" bson:"actions"`
	Active      bool               `json:"active" bson:"active"`
	LastRun     *time.Time         `json:"last_run,omitempty" bson:"last_run,omitempty"`
	NextRun     *time.Time         `json:"next_run,omitempty" bson:"next_run,omitempty"`
	CreatedBy   primitive.ObjectID `json:"created_by" bson:"created_by"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// CronJobLog represents a single execution of a cron job
type CronJobLog struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CronJobID        primitive.ObjectID `json:"cron_job_id" bson:"cron_job_id"`
	CronJobName      string             `json:"cron_job_name" bson:"cron_job_name"`
	StartTime        time.Time          `json:"start_time" bson:"start_time"`
	EndTime          *time.Time         `json:"end_time,omitempty" bson:"end_time,omitempty"`
	Status           string             `json:"status" bson:"status"` // "success", "failed", "running"
	RecordsProcessed int                `json:"records_processed" bson:"records_processed"`
	RecordsAffected  int                `json:"records_affected" bson:"records_affected"`
	Error            string             `json:"error,omitempty" bson:"error,omitempty"`
	Output           string             `json:"output,omitempty" bson:"output,omitempty"`
	CreatedAt        time.Time          `json:"created_at" bson:"created_at"`
}
