package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ContextKey string

const (
	TenantIDKey ContextKey = "tenant_id"
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
	TenantID  primitive.ObjectID `bson:"tenant_id,omitempty" json:"tenant_id,omitempty"`
	Action    AuditAction        `bson:"action" json:"action"`
	Module    string             `bson:"module" json:"module"`                       // The module/collection name
	RecordID  string             `bson:"record_id" json:"record_id"`                 // The ID of the record being modified
	ActorID   string             `bson:"actor_id" json:"actor_id"`                   // User ID who performed the action
	ActorName string             `bson:"-" json:"actor_name,omitempty"`              // Populated Name of the actor
	Changes   map[string]Change  `bson:"changes,omitempty" json:"changes,omitempty"` // For updates: field -> {old, new}
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`
}

// Product Types
type Product string

const (
	ProductCRM       Product = "crm"
	ProductERP       Product = "erp"
	ProductAnalytics Product = "analytics"
	ProductReporting Product = "reporting"
)

// Field Definitions (Moved from Module)
type FieldType string

const (
	FieldTypeText        FieldType = "text"
	FieldTypeNumber      FieldType = "number"
	FieldTypeDate        FieldType = "date"
	FieldTypeBoolean     FieldType = "boolean"
	FieldTypeLookup      FieldType = "lookup"
	FieldTypeEmail       FieldType = "email"
	FieldTypePhone       FieldType = "phone"
	FieldTypeFile        FieldType = "file"
	FieldTypeURL         FieldType = "url"
	FieldTypeTextArea    FieldType = "textarea"
	FieldTypeSelect      FieldType = "select"
	FieldTypeMultiSelect FieldType = "multiselect"
	FieldTypeCurrency    FieldType = "currency"
	FieldTypeImage       FieldType = "image"
)

type SelectOptions struct {
	Label string `json:"label" bson:"label"`
	Value string `json:"value" bson:"value"`
}

type LookupDef struct {
	LookupModule string `json:"lookup_module" bson:"lookup_module"` // Target Entity/Module Name
	LookupLabel  string `json:"lookup_label" bson:"lookup_label"`   // Target Field to display in UI
	ValueField   string `json:"value_field" bson:"value_field"`     // Target Field to store
}

type ModuleField struct {
	Name     string          `json:"name" bson:"name"`
	Label    string          `json:"label" bson:"label"`
	Type     FieldType       `json:"type" bson:"type"`
	Required bool            `json:"required" bson:"required"`
	Options  []SelectOptions `json:"options,omitempty" bson:"options,omitempty"`
	Lookup   *LookupDef      `json:"lookup,omitempty" bson:"lookup,omitempty"`
	IsSystem bool            `json:"is_system" bson:"is_system"`
}

// Entity (formerly Module) - Metadata Definition
type Entity struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID  primitive.ObjectID `json:"tenant_id" bson:"tenant_id"`
	Product   Product            `json:"product" bson:"product"`
	Name      string             `json:"name" bson:"name"` // Slug/Internal Name
	Label     string             `json:"label" bson:"label"`
	Slug      string             `json:"slug" bson:"slug"`
	Fields    []ModuleField      `json:"fields" bson:"fields"`
	Indexes   []string           `json:"indexes" bson:"indexes"`
	IsSystem  bool               `json:"is_system" bson:"is_system"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

// EntityRecord - The actual data
type EntityRecord struct {
	ID        primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	TenantID  primitive.ObjectID     `json:"tenant_id" bson:"tenant_id"`
	Product   Product                `json:"product" bson:"product"`
	Entity    string                 `json:"entity" bson:"entity"` // Name of the Entity
	Data      map[string]interface{} `json:"data" bson:"data"`
	CreatedBy string                 `json:"created_by" bson:"created_by"` // User ID
	UpdatedBy string                 `json:"updated_by" bson:"updated_by"` // User ID
	CreatedAt time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" bson:"updated_at"`
	Deleted   bool                   `json:"__deleted" bson:"deleted"`
	DeletedAt *time.Time             `json:"deleted_at,omitempty" bson:"deleted_at,omitempty"`
	DeletedBy string                 `json:"deleted_by,omitempty" bson:"deleted_by,omitempty"` // User ID
}

type Organization struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name            string             `bson:"name" json:"name"`
	Slug            string             `bson:"slug" json:"slug"`
	Plan            string             `bson:"plan" json:"plan"` // e.g. "enterprise"
	EnabledProducts []Product          `bson:"enabled_products" json:"enabled_products"`
	OwnerID         primitive.ObjectID `bson:"owner_id" json:"owner_id"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

type User struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	TenantID  primitive.ObjectID   `bson:"tenant_id,omitempty" json:"tenant_id,omitempty"`
	Username  string               `bson:"username" json:"username"`
	Password  string               `bson:"password" json:"-"`
	Email     string               `bson:"email" json:"email"`
	FirstName string               `bson:"first_name,omitempty" json:"first_name,omitempty"`
	LastName  string               `bson:"last_name,omitempty" json:"last_name,omitempty"`
	Phone     string               `bson:"phone,omitempty" json:"phone,omitempty"`
	Status    string               `bson:"status" json:"status"`                             // active, inactive, suspended
	Roles     []primitive.ObjectID `bson:"roles" json:"roles"`                               // References to Role IDs
	Groups    []string             `bson:"groups,omitempty" json:"groups,omitempty"`         // User groups for ABAC (e.g., ["sales_team_west", "managers"])
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

// Permission DSL Structures (Shared)
type RuleType string

const (
	RuleTypeStatic   RuleType = "static"
	RuleTypeVariable RuleType = "variable"
)

type PermissionRule struct {
	Field    string      `json:"field" bson:"field"`
	Operator string      `json:"operator" bson:"operator"` // eq, ne, gt, lt, gte, lte, in, nin, contains
	Value    interface{} `json:"value" bson:"value"`
	Type     RuleType    `json:"type" bson:"type"`
}

type PermissionGroup struct {
	Operator string            `json:"operator" bson:"operator"` // "AND" | "OR"
	Rules    []PermissionRule  `json:"rules" bson:"rules"`
	Groups   []PermissionGroup `json:"groups" bson:"groups"`
}

type ActionPermission struct {
	Allowed    bool             `json:"allowed" bson:"allowed"`
	Conditions *PermissionGroup `json:"conditions,omitempty" bson:"conditions,omitempty"`
}

type Filter struct {
	Field    string      `json:"field" bson:"field"`
	Operator string      `json:"operator" bson:"operator"` // eq, ne, gt, lt, gte, lte, in, nin, contains, between, starts_with, ends_with
	Value    interface{} `json:"value" bson:"value"`
}
