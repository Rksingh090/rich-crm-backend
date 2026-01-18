package blueprint

import (
	"go-crm/internal/common/models"
	"go-crm/internal/core/action"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BlueprintAction struct {
	ID     string            `bson:"id" json:"id"`
	Type   action.ActionType `bson:"type" json:"type"`
	Config map[string]any    `bson:"config" json:"config"` // Config depends on Type
	Order  int               `bson:"order" json:"order"`
}

type TransitionTriggerType string

const (
	TriggerTypeManual    TransitionTriggerType = "manual"
	TriggerTypeCondition TransitionTriggerType = "condition" // Automatic based on criteria
)

type Transition struct {
	ID           string                `bson:"id" json:"id"`
	Name         string                `bson:"name" json:"name"`
	FromState    string                `bson:"from_state" json:"from_state"`
	ToState      string                `bson:"to_state" json:"to_state"`
	SourceHandle *string               `bson:"source_handle" json:"source_handle,omitempty"` // ID of the source handle (e.g. "source-right")
	TargetHandle *string               `bson:"target_handle" json:"target_handle,omitempty"` // ID of the target handle (e.g. "target-left")
	TriggerType  TransitionTriggerType `bson:"trigger_type" json:"trigger_type"`
	Criteria     []models.Filter       `bson:"criteria" json:"criteria"` // Conditions to trigger or allow transition
	Roles        []string              `bson:"roles" json:"roles"`       // Allowed roles (if manual)
	Before       []BlueprintAction     `bson:"before" json:"before"`
	During       []BlueprintAction     `bson:"during" json:"during"`
	After        []BlueprintAction     `bson:"after" json:"after"`
	IsCommon     bool                  `bson:"is_common" json:"is_common"`
}

type Blueprint struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID    primitive.ObjectID `bson:"tenant_id" json:"tenant_id"`
	App         models.App         `bson:"app" json:"app"`                   // crm, erp, etc.
	Module      string             `bson:"module" json:"module"`             // Target Module Name
	TargetField string             `bson:"target_field" json:"target_field"` // The select field this blueprint controls
	Name        string             `bson:"name" json:"name"`
	Active      bool               `bson:"active" json:"active"`
	Transitions []Transition       `bson:"transitions" json:"transitions"`
	Layout      map[string]any     `bson:"layout" json:"layout"` // Node positions: key=stateName, val={x,y}
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
	Deleted     bool               `bson:"__deleted" json:"__deleted"`
}

type BlueprintFilter struct {
	Module string
	Search string // Name or other fields
}
