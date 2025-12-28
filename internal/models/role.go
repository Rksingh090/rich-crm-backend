package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ModulePermission defines CRUD permissions for a specific module
type ModulePermission struct {
	Create bool `json:"create" bson:"create"`
	Read   bool `json:"read" bson:"read"`
	Update bool `json:"update" bson:"update"`
	Delete bool `json:"delete" bson:"delete"`
}

const (
	FieldPermReadWrite = "read_write"
	FieldPermReadOnly  = "read_only"
	FieldPermNone      = "none"
)

// Role represents a user role with module-level permissions
type Role struct {
	ID                primitive.ObjectID           `json:"id" bson:"_id,omitempty"`
	Name              string                       `json:"name" bson:"name"`
	Description       string                       `json:"description" bson:"description"`
	ModulePermissions map[string]ModulePermission  `json:"module_permissions" bson:"module_permissions"`
	FieldPermissions  map[string]map[string]string `json:"field_permissions" bson:"field_permissions"` // Module -> Field -> "read_write" | "read_only" | "none"
	IsSystem          bool                         `json:"is_system" bson:"is_system"`                 // Prevent deletion of system roles
	CreatedAt         time.Time                    `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time                    `json:"updated_at" bson:"updated_at"`
}

type Permission struct {
	Code        string `bson:"code" json:"code"` // e.g., "user:create"
	Description string `bson:"description" json:"description"`
}
