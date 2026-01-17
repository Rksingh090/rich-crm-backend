package group

import (
	"time"

	"go-crm/internal/common/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Group represents a user group with permissions
type Group struct {
	ID               primitive.ObjectID                            `json:"id" bson:"_id,omitempty"`
	TenantID         primitive.ObjectID                            `json:"tenant_id" bson:"tenant_id"`
	App              models.App                                    `json:"app" bson:"app"`
	Name             string                                        `json:"name" bson:"name"`
	Description      string                                        `json:"description" bson:"description"`
	Permissions      map[string]map[string]models.ActionPermission `json:"permissions" bson:"permissions"`
	FieldPermissions map[string]map[string]string                  `json:"field_permissions" bson:"field_permissions"` // Module -> Field -> "read_write" | "read_only" | "none"
	Members          []primitive.ObjectID                          `json:"members" bson:"members"`                     // User IDs
	IsSystem         bool                                          `json:"is_system" bson:"is_system"`                 // Prevent deletion of system groups
	CreatedAt        time.Time                                     `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time                                     `json:"updated_at" bson:"updated_at"`
}

type ModulePermission struct {
	Read   models.ActionPermission `json:"read" bson:"read"`
	Write  models.ActionPermission `json:"write" bson:"write"`
	Delete models.ActionPermission `json:"delete" bson:"delete"`
}
