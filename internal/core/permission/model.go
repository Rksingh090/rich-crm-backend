package permission

import (
	"time"

	"go-crm/internal/common/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Permission represents a permission assignment for a role on a specific resource
// This is a first-class entity for better management and auditing
// Resources are now referenced by ID from the Resource Registry
type Permission struct {
	ID         primitive.ObjectID                 `json:"id" bson:"_id,omitempty"`
	TenantID   primitive.ObjectID                 `json:"tenant_id" bson:"tenant_id"`
	RoleID     primitive.ObjectID                 `json:"role_id" bson:"role_id"`
	RoleName   string                             `json:"role_name,omitempty" bson:"role_name,omitempty"` // For templates
	App        models.App                         `json:"app" bson:"app"`
	ResourceID string                             `json:"resource_id" bson:"resource_id"`                     // Reference to Resource Registry (e.g., "crm.leads")
	Actions    map[string]models.ActionPermission `json:"actions" bson:"actions"`                             // Action -> Permission with conditions
	FieldRules map[string]string                  `json:"field_rules,omitempty" bson:"field_rules,omitempty"` // Field -> "read_write" | "read_only" | "none"
	CreatedAt  time.Time                          `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time                          `json:"updated_at" bson:"updated_at"`
}

// AssignResourceRequest is used to assign a resource to a role with specific actions
type AssignResourceRequest struct {
	RoleID     string                             `json:"role_id" binding:"required"`
	ResourceID string                             `json:"resource_id" binding:"required"` // Now references Resource Registry
	Actions    map[string]models.ActionPermission `json:"actions" binding:"required"`
	FieldRules map[string]string                  `json:"field_rules,omitempty"`
}

// RevokeResourceRequest is used to revoke a resource from a role
type RevokeResourceRequest struct {
	RoleID     string `json:"role_id" binding:"required"`
	ResourceID string `json:"resource_id" binding:"required"`
}
