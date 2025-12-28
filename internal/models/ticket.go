package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TicketStatus represents the status of a ticket
type TicketStatus string

const (
	TicketStatusNew      TicketStatus = "new"
	TicketStatusOpen     TicketStatus = "open"
	TicketStatusPending  TicketStatus = "pending"
	TicketStatusResolved TicketStatus = "resolved"
	TicketStatusClosed   TicketStatus = "closed"
)

// TicketPriority represents the priority level of a ticket
type TicketPriority string

const (
	TicketPriorityLow    TicketPriority = "low"
	TicketPriorityMedium TicketPriority = "medium"
	TicketPriorityHigh   TicketPriority = "high"
	TicketPriorityUrgent TicketPriority = "urgent"
)

// TicketChannel represents the channel through which the ticket was created
type TicketChannel string

const (
	TicketChannelEmail  TicketChannel = "email"
	TicketChannelChat   TicketChannel = "chat"
	TicketChannelPortal TicketChannel = "portal"
	TicketChannelPhone  TicketChannel = "phone"
)

// StatusHistoryEntry represents a status change in the ticket lifecycle
// @Description History entry for ticket status changes
type StatusHistoryEntry struct {
	Status    TicketStatus       `json:"status" bson:"status" example:"open"`
	ChangedBy primitive.ObjectID `json:"changed_by" bson:"changed_by" example:"507f1f77bcf86cd799439011"`
	ChangedAt time.Time          `json:"changed_at" bson:"changed_at" example:"2024-01-13T10:00:00Z"`
	Comment   string             `json:"comment,omitempty" bson:"comment,omitempty" example:"Ticket opened for investigation"`
}

// EscalationHistoryEntry represents an escalation event
// @Description History entry for ticket escalations
type EscalationHistoryEntry struct {
	Level       int                `json:"level" bson:"level" example:"1"`
	EscalatedTo primitive.ObjectID `json:"escalated_to" bson:"escalated_to" example:"507f1f77bcf86cd799439011"`
	EscalatedAt time.Time          `json:"escalated_at" bson:"escalated_at" example:"2024-01-14T10:00:00Z"`
	Reason      string             `json:"reason" bson:"reason" example:"SLA breach detected"`
	EscalatedBy primitive.ObjectID `json:"escalated_by,omitempty" bson:"escalated_by,omitempty" example:"507f1f77bcf86cd799439011"` // System or User
	RuleID      primitive.ObjectID `json:"rule_id,omitempty" bson:"rule_id,omitempty" example:"507f1f77bcf86cd799439011"`
}

// Ticket represents a customer support ticket
// @Description Customer support ticket with multi-channel support, SLA tracking, and escalation
type Ticket struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty" example:"507f1f77bcf86cd799439011"`
	TicketNumber string             `json:"ticket_number" bson:"ticket_number" example:"TKT-000001"` // Auto-generated unique number
	Subject      string             `json:"subject" bson:"subject" example:"Unable to login to account"`
	Description  string             `json:"description" bson:"description" example:"Customer is experiencing login issues with their account"`

	// Channel Information
	Channel         TicketChannel          `json:"channel" bson:"channel" example:"email"`
	ChannelMetadata map[string]interface{} `json:"channel_metadata,omitempty" bson:"channel_metadata,omitempty"` // Email ID, Chat session, etc.

	// Priority & SLA
	Priority        TicketPriority      `json:"priority" bson:"priority" example:"high"`
	SLAPolicyID     *primitive.ObjectID `json:"sla_policy_id,omitempty" bson:"sla_policy_id,omitempty" example:"507f1f77bcf86cd799439011"`
	DueDate         *time.Time          `json:"due_date,omitempty" bson:"due_date,omitempty" example:"2024-01-15T10:00:00Z"`                   // Resolution due date
	ResponseDueDate *time.Time          `json:"response_due_date,omitempty" bson:"response_due_date,omitempty" example:"2024-01-14T10:00:00Z"` // First response due date
	FirstResponseAt *time.Time          `json:"first_response_at,omitempty" bson:"first_response_at,omitempty" example:"2024-01-13T14:30:00Z"`

	// Status Workflow
	Status        TicketStatus         `json:"status" bson:"status" example:"open"`
	StatusHistory []StatusHistoryEntry `json:"status_history,omitempty" bson:"status_history,omitempty"`

	// Assignment
	AssignedTo    *primitive.ObjectID `json:"assigned_to,omitempty" bson:"assigned_to,omitempty" example:"507f1f77bcf86cd799439011"`
	AssignedGroup string              `json:"assigned_group,omitempty" bson:"assigned_group,omitempty" example:"Support Team"` // Team/Department

	// Customer Information
	CustomerID    *primitive.ObjectID `json:"customer_id,omitempty" bson:"customer_id,omitempty" example:"507f1f77bcf86cd799439011"`
	CustomerEmail string              `json:"customer_email" bson:"customer_email" example:"customer@example.com"`
	CustomerName  string              `json:"customer_name" bson:"customer_name" example:"John Doe"`

	// Escalation
	EscalationLevel   int                      `json:"escalation_level" bson:"escalation_level" example:"0"` // 0 = no escalation
	EscalatedTo       *primitive.ObjectID      `json:"escalated_to,omitempty" bson:"escalated_to,omitempty" example:"507f1f77bcf86cd799439011"`
	EscalationHistory []EscalationHistoryEntry `json:"escalation_history,omitempty" bson:"escalation_history,omitempty"`

	// Tags and Categories
	Tags     []string `json:"tags,omitempty" bson:"tags,omitempty" example:"bug,urgent"`
	Category string   `json:"category,omitempty" bson:"category,omitempty" example:"Technical Support"`

	// Timestamps
	CreatedAt  time.Time  `json:"created_at" bson:"created_at" example:"2024-01-13T10:00:00Z"`
	UpdatedAt  time.Time  `json:"updated_at" bson:"updated_at" example:"2024-01-13T15:30:00Z"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty" bson:"resolved_at,omitempty" example:"2024-01-14T16:00:00Z"`
	ClosedAt   *time.Time `json:"closed_at,omitempty" bson:"closed_at,omitempty" example:"2024-01-15T09:00:00Z"`
}

// SLAPolicy represents a Service Level Agreement policy
// @Description SLA policy defining response and resolution time limits
type SLAPolicy struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty" example:"507f1f77bcf86cd799439011"`
	Name        string             `json:"name" bson:"name" example:"High Priority SLA"`
	Description string             `json:"description,omitempty" bson:"description,omitempty" example:"SLA for high priority tickets"`

	// Priority Mapping
	Priority TicketPriority `json:"priority" bson:"priority" example:"high"`

	// Time Limits (in minutes)
	ResponseTime   int `json:"response_time" bson:"response_time" example:"60"`      // First response time in minutes
	ResolutionTime int `json:"resolution_time" bson:"resolution_time" example:"240"` // Resolution time in minutes

	// Business Hours
	IsBusinessHoursOnly bool                   `json:"is_business_hours_only" bson:"is_business_hours_only" example:"true"`
	BusinessHours       map[string]interface{} `json:"business_hours,omitempty" bson:"business_hours,omitempty"` // JSON config for business hours

	// Status
	IsActive  bool      `json:"is_active" bson:"is_active" example:"true"`
	CreatedAt time.Time `json:"created_at" bson:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at" example:"2024-01-13T10:00:00Z"`
}

// TicketComment represents a comment or note on a ticket
// @Description Comment or internal note on a support ticket
type TicketComment struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty" example:"507f1f77bcf86cd799439011"`
	TicketID   primitive.ObjectID `json:"ticket_id" bson:"ticket_id" example:"507f1f77bcf86cd799439011"`
	Content    string             `json:"content" bson:"content" example:"Customer has been contacted via email"`
	IsInternal bool               `json:"is_internal" bson:"is_internal" example:"false"` // Internal notes vs customer-visible comments

	// Author
	CreatedBy primitive.ObjectID `json:"created_by" bson:"created_by" example:"507f1f77bcf86cd799439011"`

	// Attachments
	Attachments []primitive.ObjectID `json:"attachments,omitempty" bson:"attachments,omitempty"` // File IDs

	// Timestamps
	CreatedAt time.Time `json:"created_at" bson:"created_at" example:"2024-01-13T11:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at" example:"2024-01-13T11:00:00Z"`
}

// EscalationRule represents an automatic escalation rule
// @Description Rule for automatic ticket escalation based on conditions
type EscalationRule struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty" example:"507f1f77bcf86cd799439011"`
	Name        string             `json:"name" bson:"name" example:"High Priority No Response"`
	Description string             `json:"description,omitempty" bson:"description,omitempty" example:"Escalate high priority tickets with no response after 30 minutes"`

	// Conditions
	Priority      *TicketPriority `json:"priority,omitempty" bson:"priority,omitempty" example:"high"` // Apply to specific priority
	Status        *TicketStatus   `json:"status,omitempty" bson:"status,omitempty" example:"open"`     // Apply to specific status
	EscalateAfter int             `json:"escalate_after" bson:"escalate_after" example:"30"`           // Minutes after creation/last update
	ConditionType string          `json:"condition_type" bson:"condition_type" example:"no_response"`  // "sla_breach", "no_response", "no_update"

	// Escalation Action
	EscalateTo     primitive.ObjectID `json:"escalate_to" bson:"escalate_to" example:"507f1f77bcf86cd799439011"` // User or Group ID
	EscalateToType string             `json:"escalate_to_type" bson:"escalate_to_type" example:"user"`           // "user" or "group"
	NotifyEmails   []string           `json:"notify_emails,omitempty" bson:"notify_emails,omitempty" example:"manager@example.com,support@example.com"`

	// Status
	IsActive  bool      `json:"is_active" bson:"is_active" example:"true"`
	CreatedAt time.Time `json:"created_at" bson:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at" example:"2024-01-13T10:00:00Z"`
}
