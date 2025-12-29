package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Webhook represents a URL subscription for specific events
// @Description Webhook configuration for event notifications
type Webhook struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty" example:"507f1f77bcf86cd799439011"`
	URL         string             `json:"url" bson:"url" example:"https://example.com/webhook"`
	Secret      string             `json:"secret,omitempty" bson:"secret,omitempty" example:"my_secret_key"` // For HMCA signature
	Events      []string           `json:"events" bson:"events" example:"record.updated,record.created"`
	ModuleName  string             `json:"module_name,omitempty" bson:"module_name,omitempty" example:"leads"` // Optional: limit to specific module
	Headers     map[string]string  `json:"headers,omitempty" bson:"headers,omitempty"`                         // Custom headers to send
	IsActive    bool               `json:"is_active" bson:"is_active" example:"true"`
	Description string             `json:"description,omitempty" bson:"description,omitempty" example:"Sync contacts to remote CRM"`

	CreatedBy primitive.ObjectID `json:"created_by" bson:"created_by" example:"507f1f77bcf86cd799439011"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

// WebhookPayload represents the data sent to the webhook URL
type WebhookPayload struct {
	Event     string                 `json:"event"`
	Module    string                 `json:"module,omitempty"`
	RecordID  string                 `json:"record_id,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"` // The record data or changes
	Timestamp time.Time              `json:"timestamp"`
}

// WebhookLog represents a single execution of a webhook
type WebhookLog struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	WebhookID  primitive.ObjectID `json:"webhook_id" bson:"webhook_id"`
	URL        string             `json:"url" bson:"url"`
	Event      string             `json:"event" bson:"event"`
	Request    WebhookPayload     `json:"request" bson:"request"`
	Response   string             `json:"response,omitempty" bson:"response,omitempty"` // Body or error message
	StatusCode int                `json:"status_code" bson:"status_code"`
	Success    bool               `json:"success" bson:"success"`
	Duration   int64              `json:"duration" bson:"duration"` // Duration in milliseconds
	CreatedAt  time.Time          `json:"created_at" bson:"created_at"`
}
