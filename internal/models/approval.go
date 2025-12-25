package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ApprovalStatus defines the status of a record's approval
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
	ApprovalStatusDraft    ApprovalStatus = "draft"
)

// ApprovalWorkflow defines the rules for approving records in a module
type ApprovalWorkflow struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ModuleID  string             `bson:"module_id" json:"module_id"` // The module this workflow applies to
	Name      string             `bson:"name" json:"name"`
	Active    bool               `bson:"active" json:"active"`
	Steps     []ApprovalStep     `bson:"steps" json:"steps"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// ApprovalStep defines a single step in the approval process
type ApprovalStep struct {
	ID            string   `bson:"id" json:"id"`                         // Unique ID for the step (e.g., uuid)
	Name          string   `bson:"name" json:"name"`                     // Display name (e.g., "Manager Approval")
	Order         int      `bson:"order" json:"order"`                   // Sequence number
	ApproverRoles []string `bson:"approver_roles" json:"approver_roles"` // Role IDs allowed to approve
	ApproverUsers []string `bson:"approver_users" json:"approver_users"` // User IDs allowed to approve
}

// ApprovalRecordState tracks the approval status of a specific record
// This is embedded in the Record document's "_approval" field
type ApprovalRecordState struct {
	Status      ApprovalStatus    `bson:"status" json:"status"`
	CurrentStep int               `bson:"current_step" json:"current_step"` // Index of the current step in the workflow
	WorkflowID  string            `bson:"workflow_id" json:"workflow_id"`   // ID of the workflow definition used
	History     []ApprovalHistory `bson:"history" json:"history"`
}

// ApprovalHistory records actions taken on the record
type ApprovalHistory struct {
	StepName  string         `bson:"step_name" json:"step_name"`
	ActorID   string         `bson:"actor_id" json:"actor_id"` // User who approved/rejected
	Action    ApprovalStatus `bson:"action" json:"action"`     // approved/rejected
	Comment   string         `bson:"comment" json:"comment"`
	Timestamp time.Time      `bson:"timestamp" json:"timestamp"`
}
