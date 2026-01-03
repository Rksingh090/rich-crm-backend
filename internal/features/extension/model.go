package extension

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ExtensionStatus string

const (
	ExtensionStatusActive   ExtensionStatus = "active"
	ExtensionStatusInactive ExtensionStatus = "inactive"
	ExtensionStatusPending  ExtensionStatus = "pending"
)

type ExtensionCapability string

const (
	CapabilityWidget  ExtensionCapability = "widget"
	CapabilityWebhook ExtensionCapability = "webhook"
	CapabilityAPI     ExtensionCapability = "api"
)

type Extension struct {
	ID           primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	Name         string                 `json:"name" bson:"name"`
	Description  string                 `json:"description" bson:"description"`
	Publisher    string                 `json:"publisher" bson:"publisher"`
	Version      string                 `json:"version" bson:"version"`
	Icon         string                 `json:"icon" bson:"icon"`
	Capabilities []ExtensionCapability  `json:"capabilities" bson:"capabilities"`
	BaseURL      string                 `json:"base_url" bson:"base_url"`
	WidgetURL    string                 `json:"widget_url" bson:"widget_url"`
	WebhookURL   string                 `json:"webhook_url" bson:"webhook_url"`
	Settings     map[string]interface{} `json:"settings" bson:"settings"`
	Status       ExtensionStatus        `json:"status" bson:"status"`
	Permissions  []string               `json:"permissions" bson:"permissions"`
	Installed    bool                   `json:"installed" bson:"installed"`
	InstalledAt  *time.Time             `json:"installed_at" bson:"installed_at,omitempty"`
	CreatedAt    time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" bson:"updated_at"`
}
