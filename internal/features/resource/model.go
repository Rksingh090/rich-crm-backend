package resource

import (
	"time"

	"go-crm/internal/common/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ResourceUI struct {
	Sidebar    bool   `bson:"sidebar" json:"sidebar"`
	Order      int    `bson:"order" json:"order"`
	Group      string `bson:"group" json:"group"`
	GroupOrder int    `bson:"group_order" json:"group_order"`
	Location   string `bson:"location" json:"location"`
}

type Resource struct {
	ID             primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	ResourceID     string               `bson:"resource" json:"resource_id"`
	TenantID       primitive.ObjectID   `bson:"tenant_id,omitempty" json:"tenant_id,omitempty"` // Empty for global resources
	Product        string               `bson:"product" json:"product"`
	Type           string               `bson:"type" json:"type"`
	Key            string               `bson:"key" json:"key"`
	Label          string               `bson:"label" json:"label"`
	Icon           string               `bson:"icon" json:"icon"`
	Route          string               `bson:"route" json:"route"`
	Actions        []string             `bson:"actions" json:"actions"`
	Fields         []models.ModuleField `bson:"fields" json:"fields"`
	Configurable   bool                 `bson:"configurable" json:"configurable"`
	IsSystem       bool                 `bson:"is_system" json:"is_system"`
	Scope          string               `bson:"scope" json:"scope"`                                           // "global" or "tenant"
	IsOverride     bool                 `bson:"is_override" json:"is_override"`                               // True if this is a tenant override
	BaseResourceID string               `bson:"base_resource_id,omitempty" json:"base_resource_id,omitempty"` // For overrides, points to global resource
	UI             ResourceUI           `bson:"ui" json:"ui"`

	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
	DeletedBy string     `bson:"deleted_by,omitempty" json:"deleted_by,omitempty"`
}
