package email_template

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EmailTemplate struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID `json:"tenant_id" bson:"tenant_id"`
	App         string             `json:"app" bson:"app"`
	Name        string             `json:"name" bson:"name"`
	Subject     string             `json:"subject" bson:"subject"`
	Body        string             `json:"body" bson:"body"`
	Design      []any              `json:"design" bson:"design"`
	ModuleName  string             `json:"module_name" bson:"module_name"`
	Description string             `json:"description" bson:"description"`
	IsActive    bool               `json:"is_active" bson:"is_active"`
	CreatedBy   string             `json:"created_by" bson:"created_by"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}
