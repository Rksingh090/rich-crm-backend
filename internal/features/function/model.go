package function

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FunctionLanguage represents the scripting language for the function
type FunctionLanguage string

const (
	LanguageTengo      FunctionLanguage = "tengo"
	LanguageJavaScript FunctionLanguage = "javascript"
)

// FunctionParameter defines an input argument for the function
type FunctionParameter struct {
	Name        string `json:"name" bson:"name"`
	Type        string `json:"type" bson:"type"` // string, number, boolean, object, array, any
	Description string `json:"description" bson:"description"`
}

// Function represents a reusable script function
type Function struct {
	ID          primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID  `json:"tenant_id" bson:"tenant_id"`
	App         string              `json:"app" bson:"app"`
	Name        string              `json:"name" bson:"name"`
	Description string              `json:"description" bson:"description"`
	Language    FunctionLanguage    `json:"language" bson:"language"` // "tengo" or "javascript"
	Parameters  []FunctionParameter `json:"parameters" bson:"parameters"`
	Code        string              `json:"code" bson:"code"`
	ModuleName  string              `json:"module_name" bson:"module_name"` // Empty for global
	IsActive    bool                `json:"is_active" bson:"is_active"`
	CreatedBy   string              `json:"created_by" bson:"created_by"`
	CreatedAt   time.Time           `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at" bson:"updated_at"`
}
