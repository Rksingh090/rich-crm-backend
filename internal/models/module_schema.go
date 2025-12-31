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

type File struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	OriginalFilename string             `json:"original_filename" bson:"original_filename"`
	UniqueFilename   string             `json:"unique_filename" bson:"unique_filename"`
	Path             string             `json:"path" bson:"path"` // Server file path
	URL              string             `json:"url" bson:"url"`   // Public Web URL
	Group            string             `json:"group,omitempty" bson:"group,omitempty"`
	Size             int64              `json:"size" bson:"size"`
	MIMEType         string             `json:"mime_type" bson:"mime_type"`

	// File sharing fields
	ModuleName  string             `json:"module_name,omitempty" bson:"module_name,omitempty"`
	RecordID    string             `json:"record_id,omitempty" bson:"record_id,omitempty"`
	UploadedBy  primitive.ObjectID `json:"uploaded_by" bson:"uploaded_by"`
	IsShared    bool               `json:"is_shared" bson:"is_shared"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}
