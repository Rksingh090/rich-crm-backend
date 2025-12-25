package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Report represents a saved report configuration
type Report struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	ModuleID    string             `json:"module_id" bson:"module_id"` // The module this report is based on
	Columns     []string           `json:"columns" bson:"columns"`     // List of field names to display
	Filters     map[string]any     `json:"filters" bson:"filters"`     // Stored query filters
	CreatedBy   primitive.ObjectID `json:"created_by,omitempty" bson:"created_by,omitempty"`
	UpdatedBy   primitive.ObjectID `json:"updated_by,omitempty" bson:"updated_by,omitempty"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}
