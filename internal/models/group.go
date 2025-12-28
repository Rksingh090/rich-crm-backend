package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Group represents a user group with permissions
type Group struct {
	ID                primitive.ObjectID          `json:"id" bson:"_id,omitempty"`
	Name              string                      `json:"name" bson:"name"`
	Description       string                      `json:"description" bson:"description"`
	ModulePermissions map[string]ModulePermission `json:"module_permissions" bson:"module_permissions"`
	Members           []primitive.ObjectID        `json:"members" bson:"members"`     // User IDs
	IsSystem          bool                        `json:"is_system" bson:"is_system"` // Prevent deletion of system groups
	CreatedAt         time.Time                   `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time                   `json:"updated_at" bson:"updated_at"`
}
