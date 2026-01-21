package resource

import (
	"time"

	"go-crm/internal/common/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ResourceScope string

const (
	ResourceScopeGlobal ResourceScope = "global" // Available to all tenants
	ResourceScopeApp    ResourceScope = "app"    // Available to all tenants using this app
	ResourceScopeTenant ResourceScope = "tenant" // Tenant-specific custom resource
)

type ResourceType string

const (
	ResourceTypeModule  ResourceType = "module"
	ResourceTypePage    ResourceType = "page"
	ResourceTypeSetting ResourceType = "setting"
	ResourceTypeSystem  ResourceType = "system"
	ResourceTypeCron    ResourceType = "cron"
	ResourceTypeWebhook ResourceType = "webhook"
)

// Resource represents a discoverable resource in the system
// Resources can be global (available to all), app-level (available to all tenants using an app),
// or tenant-specific (custom resources created by a tenant)
type Resource struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ResourceID string              `bson:"resource_id" json:"resource_id"` // e.g., "crm.leads", "erp.inventory"
	App        models.App          `bson:"app" json:"app"`
	TenantID   *primitive.ObjectID `bson:"tenant_id,omitempty" json:"tenant_id,omitempty"` // null for global/app-level

	Type  ResourceType `bson:"type" json:"type"`
	Key   string       `bson:"key" json:"key"`     // Internal key (e.g., "leads")
	Label string       `bson:"label" json:"label"` // Display name (e.g., "Leads")
	Icon  string       `bson:"icon,omitempty" json:"icon,omitempty"`
	Route string       `bson:"route,omitempty" json:"route,omitempty"`

	// Available actions for this resource
	AvailableActions []string `bson:"available_actions" json:"available_actions"` // ["read", "create", "update", "delete"]

	// Scope determines visibility
	Scope          ResourceScope `bson:"scope" json:"scope"`
	IsSystem       bool          `bson:"is_system" json:"is_system"`             // System resources cannot be deleted
	IsConfigurable bool          `bson:"is_configurable" json:"is_configurable"` // Can permissions be configured

	// UI metadata for rendering in frontend
	UI ResourceUI `bson:"ui,omitempty" json:"ui,omitempty"`

	// Lifecycle
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// ResourceUI contains UI-specific metadata for rendering resources
type ResourceUI struct {
	Sidebar    bool   `bson:"sidebar" json:"sidebar"`
	Location   string `bson:"location,omitempty" json:"location,omitempty"` // "main", "settings"
	Group      string `bson:"group,omitempty" json:"group,omitempty"`
	GroupOrder int    `bson:"group_order,omitempty" json:"group_order,omitempty"`
	Order      int    `bson:"order" json:"order"`
}

// ResourceFilter is used for querying resources
type ResourceFilter struct {
	TenantID      *primitive.ObjectID
	App           *models.App
	Type          *ResourceType
	Scope         *ResourceScope
	IncludeGlobal bool // Include global resources in tenant queries
}
