package permission

import (
	"time"

	"go-crm/internal/common/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ResourceRef identifies a resource (module, page, setting, etc.)
type ResourceRef struct {
	Type string `json:"type" bson:"type"` // "module", "page", "setting", "report", "cron", etc.
	ID   string `json:"id" bson:"id"`     // Resource identifier (e.g., "crm.leads", "crm.settings")
}

// Permission represents a permission assignment for a role on a specific resource
// This is a first-class entity for better management and auditing
type Permission struct {
	ID         primitive.ObjectID                 `json:"id" bson:"_id,omitempty"`
	TenantID   primitive.ObjectID                 `json:"tenant_id" bson:"tenant_id"`
	RoleID     primitive.ObjectID                 `json:"role_id" bson:"role_id"`
	Resource   ResourceRef                        `json:"resource" bson:"resource"`
	Actions    map[string]models.ActionPermission `json:"actions" bson:"actions"`                             // Action -> Permission with conditions
	FieldRules map[string]string                  `json:"field_rules,omitempty" bson:"field_rules,omitempty"` // Field -> "read_write" | "read_only" | "none"
	CreatedAt  time.Time                          `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time                          `json:"updated_at" bson:"updated_at"`
}

// AssignResourceRequest is used to assign a resource to a role with specific actions
type AssignResourceRequest struct {
	RoleID     string                             `json:"role_id" binding:"required"`
	ResourceID string                             `json:"resource_id" binding:"required"`
	Actions    map[string]models.ActionPermission `json:"actions" binding:"required"`
	FieldRules map[string]string                  `json:"field_rules,omitempty"`
}

// RevokeResourceRequest is used to revoke a resource from a role
type RevokeResourceRequest struct {
	RoleID     string `json:"role_id" binding:"required"`
	ResourceID string `json:"resource_id" binding:"required"`
}
