package saved_filter

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SavedFilter represents a saved filter configuration
type SavedFilter struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	ModuleName  string             `json:"module_name" bson:"module_name"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	TenantID    primitive.ObjectID `json:"tenant_id" bson:"tenant_id"`
	IsPublic    bool               `json:"is_public" bson:"is_public"`
	IsDefault   bool               `json:"is_default" bson:"is_default"`
	Criteria    FilterCriteria     `json:"criteria" bson:"criteria"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// FilterCriteria represents filter logic with conditions
type FilterCriteria struct {
	Logic      string            `json:"logic" bson:"logic"` // "AND" or "OR"
	Conditions []FilterCondition `json:"conditions" bson:"conditions"`
	Groups     []FilterCriteria  `json:"groups,omitempty" bson:"groups,omitempty"` // Nested groups
}

// FilterCondition represents a single filter condition
type FilterCondition struct {
	Field    string      `json:"field" bson:"field"`
	Operator string      `json:"operator" bson:"operator"` // eq, ne, gt, lt, gte, lte, contains, in, etc.
	Value    interface{} `json:"value" bson:"value"`
}
