package models

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
}

type LookupDef struct {
	Module       string `json:"module" bson:"module"`               // Target Module Name
	DisplayField string `json:"display_field" bson:"display_field"` // Target Field to display in UI
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

type File struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	OriginalFilename string             `json:"original_filename" bson:"original_filename"`
	UniqueFilename   string             `json:"unique_filename" bson:"unique_filename"`
	Path             string             `json:"path" bson:"path"` // Server file path
	URL              string             `json:"url" bson:"url"`   // Public Web URL
	Group            string             `json:"group,omitempty" bson:"group,omitempty"`
	Size             int64              `json:"size" bson:"size"`
	MIMEType         string             `json:"mime_type" bson:"mime_type"`
	CreatedAt        time.Time          `json:"created_at" bson:"created_at"`
}
