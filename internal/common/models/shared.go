package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuditAction string

const (
	AuditActionCreate     AuditAction = "CREATE"
	AuditActionUpdate     AuditAction = "UPDATE"
	AuditActionDelete     AuditAction = "DELETE"
	AuditActionLogin      AuditAction = "LOGIN"
	AuditActionAutomation AuditAction = "AUTOMATION"
	AuditActionApproval   AuditAction = "APPROVAL"
	AuditActionSync       AuditAction = "SYNC"
	AuditActionCron       AuditAction = "CRON"
	AuditActionSettings   AuditAction = "SETTINGS"
	AuditActionTemplate   AuditAction = "TEMPLATE"
	AuditActionWebhook    AuditAction = "WEBHOOK"
	AuditActionGroup      AuditAction = "GROUP"
	AuditActionReport     AuditAction = "REPORT"
	AuditActionChart      AuditAction = "CHART"
	AuditActionDashboard  AuditAction = "DASHBOARD"
)

type Change struct {
	Old interface{} `bson:"old" json:"old"`
	New interface{} `bson:"new" json:"new"`
}

type AuditLog struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Action    AuditAction        `bson:"action" json:"action"`
	Module    string             `bson:"module" json:"module"`                       // The module/collection name
	RecordID  string             `bson:"record_id" json:"record_id"`                 // The ID of the record being modified
	ActorID   string             `bson:"actor_id" json:"actor_id"`                   // User ID who performed the action
	ActorName string             `bson:"-" json:"actor_name,omitempty"`              // Populated Name of the actor
	Changes   map[string]Change  `bson:"changes,omitempty" json:"changes,omitempty"` // For updates: field -> {old, new}
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`
}

type User struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Username  string               `bson:"username" json:"username"`
	Password  string               `bson:"password" json:"-"`
	Email     string               `bson:"email" json:"email"`
	FirstName string               `bson:"first_name,omitempty" json:"first_name,omitempty"`
	LastName  string               `bson:"last_name,omitempty" json:"last_name,omitempty"`
	Phone     string               `bson:"phone,omitempty" json:"phone,omitempty"`
	Status    string               `bson:"status" json:"status"`                             // active, inactive, suspended
	Roles     []primitive.ObjectID `bson:"roles" json:"roles"`                               // References to Role IDs
	ReportsTo *primitive.ObjectID  `bson:"reports_to,omitempty" json:"reports_to,omitempty"` // Manager ID
	LastLogin *time.Time           `bson:"last_login,omitempty" json:"last_login,omitempty"`
	CreatedAt time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time            `bson:"updated_at" json:"updated_at"`
}

type Log struct {
	Message      string    `bson:"message" json:"message"`
	IpAddress    string    `bson:"ip_address" json:"ip_address"` // Actual IP
	CustomerId   int       `bson:"customer_id" json:"customer_id"`
	LogLevelId   int       `bson:"log_level_id" json:"log_level_id"`
	CreatedOnUtc time.Time `bson:"created_on_utc" json:"created_on_utc"`
}

type WebhookPayload struct {
	Event     string         `json:"event"`
	Module    string         `json:"module,omitempty"`
	RecordID  string         `json:"record_id,omitempty"`
	Data      interface{}    `json:"data"`
	Timestamp time.Time      `json:"timestamp"`
	Extra     map[string]any `json:"extra,omitempty"`
}

// Rule Models
type RuleCondition struct {
	Field    string      `json:"field" bson:"field"`
	Operator string      `json:"operator" bson:"operator"`
	Value    interface{} `json:"value" bson:"value"`
}

type RuleAction struct {
	Type   string                 `json:"type" bson:"type"`
	Config map[string]interface{} `json:"config" bson:"config"`
}

// Approval Models
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
	ApprovalStatusDraft    ApprovalStatus = "draft"
)

type ApprovalRecordState struct {
	Status      ApprovalStatus    `bson:"status" json:"status"`
	CurrentStep int               `bson:"current_step" json:"current_step"`
	WorkflowID  string            `bson:"workflow_id" json:"workflow_id"`
	History     []ApprovalHistory `bson:"history" json:"history"`
}

type ApprovalHistory struct {
	StepName  string         `bson:"step_name" json:"step_name"`
	ActorID   string         `bson:"actor_id" json:"actor_id"`
	Action    ApprovalStatus `bson:"action" json:"action"`
	Comment   string         `bson:"comment" json:"comment"`
	Timestamp time.Time      `bson:"timestamp" json:"timestamp"`
}
