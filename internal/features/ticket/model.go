package ticket

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
type StatusHistoryEntry struct {
	Status    TicketStatus       `json:"status" bson:"status"`
	ChangedBy primitive.ObjectID `json:"changed_by" bson:"changed_by"`
	ChangedAt time.Time          `json:"changed_at" bson:"changed_at"`
	Comment   string             `json:"comment,omitempty" bson:"comment,omitempty"`
}

// EscalationHistoryEntry represents an escalation event
type EscalationHistoryEntry struct {
	Level       int                `json:"level" bson:"level"`
	EscalatedTo primitive.ObjectID `json:"escalated_to" bson:"escalated_to"`
	EscalatedAt time.Time          `json:"escalated_at" bson:"escalated_at"`
	Reason      string             `json:"reason" bson:"reason"`
	EscalatedBy primitive.ObjectID `json:"escalated_by,omitempty" bson:"escalated_by,omitempty"`
	RuleID      primitive.ObjectID `json:"rule_id,omitempty" bson:"rule_id,omitempty"`
}

// Ticket represents a customer support ticket
type Ticket struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TicketNumber string             `json:"ticket_number" bson:"ticket_number"`
	Subject      string             `json:"subject" bson:"subject"`
	Description  string             `json:"description" bson:"description"`

	// Channel Information
	Channel         TicketChannel          `json:"channel" bson:"channel"`
	ChannelMetadata map[string]interface{} `json:"channel_metadata,omitempty" bson:"channel_metadata,omitempty"`

	// Priority & SLA
	Priority        TicketPriority      `json:"priority" bson:"priority"`
	SLAPolicyID     *primitive.ObjectID `json:"sla_policy_id,omitempty" bson:"sla_policy_id,omitempty"`
	DueDate         *time.Time          `json:"due_date,omitempty" bson:"due_date,omitempty"`
	ResponseDueDate *time.Time          `json:"response_due_date,omitempty" bson:"response_due_date,omitempty"`
	FirstResponseAt *time.Time          `json:"first_response_at,omitempty" bson:"first_response_at,omitempty"`

	// Status Workflow
	Status        TicketStatus         `json:"status" bson:"status"`
	StatusHistory []StatusHistoryEntry `json:"status_history,omitempty" bson:"status_history,omitempty"`

	// Assignment
	AssignedTo    *primitive.ObjectID `json:"assigned_to,omitempty" bson:"assigned_to,omitempty"`
	AssignedGroup string              `json:"assigned_group,omitempty" bson:"assigned_group,omitempty"`

	// Customer Information
	CustomerID    *primitive.ObjectID `json:"customer_id,omitempty" bson:"customer_id,omitempty"`
	CustomerEmail string              `json:"customer_email" bson:"customer_email"`
	CustomerName  string              `json:"customer_name" bson:"customer_name"`

	// Escalation
	EscalationLevel   int                      `json:"escalation_level" bson:"escalation_level"`
	EscalatedTo       *primitive.ObjectID      `json:"escalated_to,omitempty" bson:"escalated_to,omitempty"`
	EscalationHistory []EscalationHistoryEntry `json:"escalation_history,omitempty" bson:"escalation_history,omitempty"`

	// Tags and Categories
	Tags     []string `json:"tags,omitempty" bson:"tags,omitempty"`
	Category string   `json:"category,omitempty" bson:"category,omitempty"`

	// Timestamps
	CreatedAt  time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" bson:"updated_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty" bson:"resolved_at,omitempty"`
	ClosedAt   *time.Time `json:"closed_at,omitempty" bson:"closed_at,omitempty"`
}

// SLAPolicy represents a Service Level Agreement policy
type SLAPolicy struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`

	// Priority Mapping
	Priority TicketPriority `json:"priority" bson:"priority"`

	// Time Limits (in minutes)
	ResponseTime   int `json:"response_time" bson:"response_time"`
	ResolutionTime int `json:"resolution_time" bson:"resolution_time"`

	// Business Hours
	IsBusinessHoursOnly bool                   `json:"is_business_hours_only" bson:"is_business_hours_only"`
	BusinessHours       map[string]interface{} `json:"business_hours,omitempty" bson:"business_hours,omitempty"`

	// Status
	IsActive  bool      `json:"is_active" bson:"is_active"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// TicketComment represents a comment or note on a ticket
type TicketComment struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TicketID   primitive.ObjectID `json:"ticket_id" bson:"ticket_id"`
	Content    string             `json:"content" bson:"content"`
	IsInternal bool               `json:"is_internal" bson:"is_internal"`

	// Author
	CreatedBy primitive.ObjectID `json:"created_by" bson:"created_by"`

	// Attachments
	Attachments []primitive.ObjectID `json:"attachments,omitempty" bson:"attachments,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// EscalationRule represents an automatic escalation rule
type EscalationRule struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`

	// Conditions
	Priority      *TicketPriority `json:"priority,omitempty" bson:"priority,omitempty"`
	Status        *TicketStatus   `json:"status,omitempty" bson:"status,omitempty"`
	EscalateAfter int             `json:"escalate_after" bson:"escalate_after"`
	ConditionType string          `json:"condition_type" bson:"condition_type"`

	// Escalation Action
	EscalateTo     primitive.ObjectID `json:"escalate_to" bson:"escalate_to"`
	EscalateToType string             `json:"escalate_to_type" bson:"escalate_to_type"`
	NotifyEmails   []string           `json:"notify_emails,omitempty" bson:"notify_emails,omitempty"`

	// Status
	IsActive  bool      `json:"is_active" bson:"is_active"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}
