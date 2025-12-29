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
	AuditActionWorkflow   AuditAction = "WORKFLOW"
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
