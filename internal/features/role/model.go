package role

import (
	"time"

	"go-crm/internal/common/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ModulePermission defines CRUD permissions for a specific module
type ModulePermission struct {
	Create models.ActionPermission `json:"create" bson:"create"`
	Read   models.ActionPermission `json:"read" bson:"read"`
	Update models.ActionPermission `json:"update" bson:"update"`
	Delete models.ActionPermission `json:"delete" bson:"delete"`
}

const (
	FieldPermReadWrite = "read_write"
	FieldPermReadOnly  = "read_only"
	FieldPermNone      = "none"
)

// Role represents a user role with module-level permissions
type Role struct {
	ID          primitive.ObjectID                            `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID                            `json:"tenant_id" bson:"tenant_id,omitempty"`
	Name        string                                        `json:"name" bson:"name"`
	Description string                                        `json:"description" bson:"description"`
	Permissions map[string]map[string]models.ActionPermission `json:"permissions" bson:"permissions"` // Resource -> Action -> Permission
	// Legacy support alias if needed, strict mapping can double-map if tags overlap
	// But let's just rename it "Permissions" and handle migration properly.
	// For backward compat in code, "ModulePermissions" name removal implies updating all references.

	FieldPermissions map[string]map[string]string `json:"field_permissions" bson:"field_permissions"` // Module -> Field -> "read_write" | "read_only" | "none"
	IsSystem         bool                         `json:"is_system" bson:"is_system"`                 // Prevent deletion of system roles
	CreatedAt        time.Time                    `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time                    `json:"updated_at" bson:"updated_at"`
}

type Permission struct {
	Code        string `bson:"code" json:"code"` // e.g., "user:create"
	Description string `bson:"description" json:"description"`
}
