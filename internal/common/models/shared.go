package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ContextKey string

const (
	TenantIDKey ContextKey = "tenant_id"
	AppIDKey    ContextKey = "app"
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

// App Types
type App string

const (
	AppCRM       App = "crm"
	AppERP       App = "erp"
	AppAnalytics App = "analytics"
)

type AppStatus string

const (
	AppStatusActive      AppStatus = "active"
	AppStatusInactive    AppStatus = "inactive"
	AppStatusMaintenance AppStatus = "maintenance"
)

type Application struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"` // crm, erp, analytics
	Label       string             `bson:"label" json:"label"`
	Description string             `bson:"description" json:"description"`
	Icon        string             `bson:"icon" json:"icon"`
	Status      AppStatus          `bson:"status" json:"status"`
	Version     string             `bson:"version" json:"version"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

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
	FieldTypeUser        FieldType = "user"
	FieldTypeRadio       FieldType = "radio"   // New: Radio buttons
	FieldTypeSubform     FieldType = "subform" // New: Nested Table/Array
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
	Name          string          `json:"name" bson:"name"`
	Label         string          `json:"label" bson:"label"`
	Type          FieldType       `json:"type" bson:"type"`
	Required      bool            `json:"required" bson:"required"`
	Options       []SelectOptions `json:"options,omitempty" bson:"options,omitempty"`
	Lookup        *LookupDef      `json:"lookup,omitempty" bson:"lookup,omitempty"`
	SubFields     []ModuleField   `json:"sub_fields" bson:"sub_fields"` // New: Schema for subform rows - Removed omitempty to ensure persistence
	IsSystem      bool            `json:"is_system" bson:"is_system"`
	Filterable    bool            `json:"filterable" bson:"filterable"`
	Sortable      bool            `json:"sortable" bson:"sortable"`
	Unique        bool            `json:"unique" bson:"unique"`
	DefaultValue  string          `json:"default_value" bson:"default_value"`
	Placeholder   string          `json:"placeholder" bson:"placeholder"`
	HelpText      string          `json:"help_text" bson:"help_text"`
	Hidden        bool            `json:"hidden" bson:"hidden"`
	ReadOnly      bool            `json:"readonly" bson:"readonly"`
	AutoIncrement bool            `json:"auto_increment" bson:"auto_increment"` // New: Auto-increment for numbers
	SectionID     string          `json:"section_id" bson:"section_id"`         // New: Section ID
}

type Section struct {
	ID    string `bson:"id" json:"id"`
	Name  string `bson:"name" json:"name"`
	Order int    `bson:"order" json:"order"`
}

// Entity (formerly Module) - Metadata Definition
type Entity struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID     primitive.ObjectID `json:"tenant_id" bson:"tenant_id"`
	App          App                `json:"app" bson:"app"`
	Name         string             `json:"name" bson:"name"` // Slug/Internal Name
	Label        string             `json:"label" bson:"label"`
	Sections     []Section          `json:"sections" bson:"sections"` // New: Sections
	Fields       []ModuleField      `json:"fields" bson:"fields"`
	Indexes      []string           `json:"indexes" bson:"indexes"`
	IsSystem     bool               `json:"is_system" bson:"is_system"`
	Scope        string             `json:"scope" bson:"scope"`                                       // "global" or "tenant"
	IsOverride   bool               `json:"is_override" bson:"is_override"`                           // Is this a tenant override of a global entity?
	BaseEntityID primitive.ObjectID `json:"base_entity_id,omitempty" bson:"base_entity_id,omitempty"` // If Override, ID of the global entity
	CanOverride  bool               `json:"can_override" bson:"can_override"`                         // Can this entity be overridden? (for global entities)
	ReadOnly     bool               `json:"readonly" bson:"readonly"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
	DeletedAt    *time.Time         `json:"deleted_at,omitempty" bson:"deleted_at,omitempty"`
	DeletedBy    string             `json:"deleted_by,omitempty" bson:"deleted_by,omitempty"`
}

// EntityRecord - The actual data
type EntityRecord struct {
	ID        primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	TenantID  primitive.ObjectID     `json:"tenant_id" bson:"tenant_id"`
	App       App                    `json:"app" bson:"app"`
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

type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusPastDue   SubscriptionStatus = "past_due"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusTrial     SubscriptionStatus = "trial"
	SubscriptionStatusFree      SubscriptionStatus = "free"
)

type ValidationStatus string

const (
	ValidationStatusVerified   ValidationStatus = "verified"
	ValidationStatusUnverified ValidationStatus = "unverified"
	ValidationStatusPending    ValidationStatus = "pending"
)

type Organization struct {
	ID                      primitive.ObjectID                     `bson:"_id,omitempty" json:"id"`
	Name                    string                                 `bson:"name" json:"name"`
	Slug                    string                                 `bson:"slug" json:"slug"`
	Plan                    string                                 `bson:"plan" json:"plan"` // e.g. "enterprise", "pro", "free"
	SubscriptionStatus      SubscriptionStatus                     `bson:"subscription_status" json:"subscription_status"`
	BillingCycle            string                                 `bson:"billing_cycle" json:"billing_cycle"` // "monthly", "yearly"
	Currency                string                                 `bson:"currency" json:"currency"`           // "USD", "EUR"
	Price                   float64                                `bson:"price" json:"price"`
	TrialEndsAt             *time.Time                             `bson:"trial_ends_at,omitempty" json:"trial_ends_at,omitempty"`
	NextBillingDate         *time.Time                             `bson:"next_billing_date,omitempty" json:"next_billing_date,omitempty"`
	PaymentMethodID         string                                 `bson:"payment_method_id,omitempty" json:"payment_method_id,omitempty"`
	ValidationStatus        ValidationStatus                       `bson:"validation_status" json:"validation_status"`
	EnabledApps             []App                                  `bson:"enabled_apps" json:"enabled_apps"`
	OwnerID                 primitive.ObjectID                     `bson:"owner_id" json:"owner_id"`
	CreatedBy               primitive.ObjectID                     `bson:"created_by,omitempty" json:"created_by,omitempty"`
	DefaultPermissions      map[string]map[string]ActionPermission `bson:"default_permissions,omitempty" json:"default_permissions,omitempty"`             // Default actions for any user in org
	DefaultFieldPermissions map[string]map[string]string           `bson:"default_field_permissions,omitempty" json:"default_field_permissions,omitempty"` // Default field access
	CreatedAt               time.Time                              `bson:"created_at" json:"created_at"`
	UpdatedAt               time.Time                              `bson:"updated_at" json:"updated_at"`
}

const StatusInvited = "invited"

type UserAppRole struct {
	AppID  App                `bson:"app_id" json:"app_id"`
	RoleID primitive.ObjectID `bson:"role_id" json:"role_id"`
}

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID primitive.ObjectID `bson:"tenant_id,omitempty" json:"tenant_id,omitempty"`

	Password  string `bson:"password" json:"-"`
	Email     string `bson:"email" json:"email"`
	FirstName string `bson:"first_name,omitempty" json:"first_name,omitempty"`
	LastName  string `bson:"last_name,omitempty" json:"last_name,omitempty"`
	Phone     string `bson:"phone,omitempty" json:"phone,omitempty"`
	Status    string `bson:"status" json:"status"` // active, inactive, suspended, invited

	AppRoles         []UserAppRole                          `bson:"app_roles,omitempty" json:"app_roles,omitempty"`
	Groups           []primitive.ObjectID                   `bson:"groups,omitempty" json:"groups,omitempty"`                       // References to Group IDs
	Permissions      map[string]map[string]ActionPermission `bson:"permissions,omitempty" json:"permissions,omitempty"`             // Direct user overrides
	FieldPermissions map[string]map[string]string           `bson:"field_permissions,omitempty" json:"field_permissions,omitempty"` // Direct user field overrides
	ReportsTo        *primitive.ObjectID                    `bson:"reports_to,omitempty" json:"reports_to,omitempty"`               // Manager ID
	LastLogin        *time.Time                             `bson:"last_login,omitempty" json:"last_login,omitempty"`
	CreatedAt        time.Time                              `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time                              `bson:"updated_at" json:"updated_at"`

	// New fields for invite flow
	InviteToken     string     `bson:"invite_token,omitempty" json:"-"`
	InviteExpiresAt *time.Time `bson:"invite_expires_at,omitempty" json:"-"`
	IsPlatformAdmin bool       `bson:"is_platform_admin" json:"is_platform_admin"`
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

type ActionUI struct {
	Filters []string `json:"filters,omitempty" bson:"filters,omitempty"`
}

type ActionPermission struct {
	Allowed    bool             `json:"allowed" bson:"allowed"`
	Conditions *PermissionGroup `json:"conditions,omitempty" bson:"conditions,omitempty"`
	UI         *ActionUI        `json:"ui,omitempty" bson:"ui,omitempty"`
}

type Filter struct {
	Field    string      `json:"field" bson:"field"`
	Operator string      `json:"operator" bson:"operator"` // eq, ne, gt, lt, gte, lte, in, nin, contains, between, starts_with, ends_with
	Value    interface{} `json:"value" bson:"value"`
}
