package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DashboardWidget represents a single widget on a dashboard
type DashboardWidget struct {
	ID         string                 `json:"id" bson:"id"`                   // Unique widget ID
	Type       string                 `json:"type" bson:"type"`               // metric, chart, table, etc.
	Title      string                 `json:"title" bson:"title"`             // Widget title
	ModuleName string                 `json:"module_name" bson:"module_name"` // Source module
	Position   WidgetPosition         `json:"position" bson:"position"`       // Grid position
	Config     map[string]interface{} `json:"config" bson:"config"`           // Widget-specific config
}

// WidgetPosition defines the position and size of a widget in the grid
type WidgetPosition struct {
	X      int `json:"x" bson:"x"`           // X position
	Y      int `json:"y" bson:"y"`           // Y position
	Width  int `json:"width" bson:"width"`   // Width in grid units
	Height int `json:"height" bson:"height"` // Height in grid units
}

// DashboardConfig represents a user's custom dashboard configuration
type DashboardConfig struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	IsDefault   bool               `json:"is_default" bson:"is_default"` // Default dashboard for user
	IsShared    bool               `json:"is_shared" bson:"is_shared"`   // Shared with other users
	Widgets     []DashboardWidget  `json:"widgets" bson:"widgets"`       // Widget configurations
	Layout      string             `json:"layout" bson:"layout"`         // grid, vertical, etc.
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}
