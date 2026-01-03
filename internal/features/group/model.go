package group

import (
	"time"

	"go-crm/internal/common/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Group represents a user group with permissions
type Group struct {
	ID          primitive.ObjectID                            `json:"id" bson:"_id,omitempty"`
	Name        string                                        `json:"name" bson:"name"`
	Description string                                        `json:"description" bson:"description"`
	Permissions map[string]map[string]models.ActionPermission `json:"permissions" bson:"permissions"`
	Members     []primitive.ObjectID                          `json:"members" bson:"members"`     // User IDs
	IsSystem    bool                                          `json:"is_system" bson:"is_system"` // Prevent deletion of system groups
	CreatedAt   time.Time                                     `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time                                     `json:"updated_at" bson:"updated_at"`
}

type ModulePermission struct {
	Read   models.ActionPermission `json:"read" bson:"read"`
	Write  models.ActionPermission `json:"write" bson:"write"`
	Delete models.ActionPermission `json:"delete" bson:"delete"`
}
