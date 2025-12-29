package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CronJob represents a scheduled automation job
// @Description Cron job configuration for scheduled automation
type CronJob struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty" example:"507f1f77bcf86cd799439011"`
	Name        string             `json:"name" bson:"name" example:"Daily Lead Reminder"`
	Description string             `json:"description,omitempty" bson:"description,omitempty" example:"Send daily reminder for uncontacted leads"`

	// Cron expression (e.g., "0 0 * * *" for daily at midnight)
	Schedule string `json:"schedule" bson:"schedule" example:"0 0 * * *"`

	// Optional module to target for record-based operations
	ModuleID string `json:"module_id,omitempty" bson:"module_id,omitempty" example:"leads"`

	// Conditions to filter records (optional, for record-based jobs)
	Conditions []RuleCondition `json:"conditions,omitempty" bson:"conditions,omitempty"`

	// Actions to execute (reusing automation action types)
	Actions []RuleAction `json:"actions" bson:"actions"`

	// Whether the job is active
	Active bool `json:"active" bson:"active" example:"true"`

	// Execution tracking
	LastRun *time.Time `json:"last_run,omitempty" bson:"last_run,omitempty"`
	NextRun *time.Time `json:"next_run,omitempty" bson:"next_run,omitempty"`

	// Audit fields
	CreatedBy primitive.ObjectID `json:"created_by" bson:"created_by" example:"507f1f77bcf86cd799439011"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

// CronJobLog represents a single execution of a cron job
// @Description Execution log for a cron job
type CronJobLog struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CronJobID   primitive.ObjectID `json:"cron_job_id" bson:"cron_job_id"`
	CronJobName string             `json:"cron_job_name" bson:"cron_job_name"` // Denormalized for easier querying

	// Execution timing
	StartTime time.Time  `json:"start_time" bson:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty" bson:"end_time,omitempty"`

	// Execution results
	Status           string `json:"status" bson:"status"` // "success", "failed", "running"
	RecordsProcessed int    `json:"records_processed" bson:"records_processed"`
	RecordsAffected  int    `json:"records_affected" bson:"records_affected"`

	// Error and output details
	Error  string `json:"error,omitempty" bson:"error,omitempty"`
	Output string `json:"output,omitempty" bson:"output,omitempty"` // JSON or text output

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}
