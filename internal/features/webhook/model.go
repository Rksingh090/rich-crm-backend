package webhook

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Webhook represents a URL subscription for specific events
type Webhook struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID `json:"tenant_id" bson:"tenant_id"`
	URL         string             `json:"url" bson:"url"`
	Secret      string             `json:"secret,omitempty" bson:"secret,omitempty"` // For HMCA signature
	Events      []string           `json:"events" bson:"events"`
	ModuleName  string             `json:"module_name,omitempty" bson:"module_name,omitempty"` // Optional: limit to specific module
	Headers     map[string]string  `json:"headers,omitempty" bson:"headers,omitempty"`         // Custom headers to send
	IsActive    bool               `json:"is_active" bson:"is_active"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`

	CreatedBy primitive.ObjectID `json:"created_by" bson:"created_by"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

// WebhookLog represents a single execution of a webhook
type WebhookLog struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID   primitive.ObjectID `json:"tenant_id" bson:"tenant_id"`
	WebhookID  primitive.ObjectID `json:"webhook_id" bson:"webhook_id"`
	URL        string             `json:"url" bson:"url"`
	Event      string             `json:"event" bson:"event"`
	Request    any                `json:"request" bson:"request"`
	Response   string             `json:"response,omitempty" bson:"response,omitempty"` // Body or error message
	StatusCode int                `json:"status_code" bson:"status_code"`
	Success    bool               `json:"success" bson:"success"`
	Duration   int64              `json:"duration" bson:"duration"` // Duration in milliseconds
	CreatedAt  time.Time          `json:"created_at" bson:"created_at"`
}
