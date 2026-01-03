package module

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FieldType string

const (
	FieldTypeText        FieldType = "text"
	FieldTypeNumber      FieldType = "number"
	FieldTypeDate        FieldType = "date"
	FieldTypeBoolean     FieldType = "boolean"
	FieldTypeLookup      FieldType = "lookup"
	FieldTypeEmail       FieldType = "email"
	FieldTypePhone       FieldType = "phone"
	FieldTypeFile        FieldType = "file"
	FieldTypeURL         FieldType = "url"
	FieldTypeTextArea    FieldType = "textarea"
	FieldTypeSelect      FieldType = "select"
	FieldTypeMultiSelect FieldType = "multiselect"
	FieldTypeCurrency    FieldType = "currency"
	FieldTypeImage       FieldType = "image"
)

type SelectOptions struct {
	Label string `json:"label" bson:"label"`
	Value string `json:"value" bson:"value"`
}

type ModuleField struct {
	Name     string          `json:"name" bson:"name"`
	Label    string          `json:"label" bson:"label"`
	Type     FieldType       `json:"type" bson:"type"`
	Required bool            `json:"required" bson:"required"`
	Options  []SelectOptions `json:"options,omitempty" bson:"options,omitempty"` // For Select/MultiSelect
	Lookup   *LookupDef      `json:"lookup,omitempty" bson:"lookup,omitempty"`
	IsSystem bool            `json:"is_system" bson:"is_system"`
}

type LookupDef struct {
	LookupModule string `json:"lookup_module" bson:"lookup_module"` // Target Module Name
	LookupLabel  string `json:"lookup_label" bson:"lookup_label"`   // Target Field to display in UI (e.g. name)
	ValueField   string `json:"value_field" bson:"value_field"`     // Target Field to store (usually _id)
}

type Module struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name      string             `json:"name" bson:"name"` // Unique Identifier (e.g., "leads", "deals")
	Label     string             `json:"label" bson:"label"`
	IsSystem  bool               `json:"is_system" bson:"is_system"` // If true, cannot be deleted
	Fields    []ModuleField      `json:"fields" bson:"fields"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}
