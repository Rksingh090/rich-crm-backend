package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BulkOperationStatus string

const (
	BulkStatusPending    BulkOperationStatus = "pending"
	BulkStatusProcessing BulkOperationStatus = "processing"
	BulkStatusCompleted  BulkOperationStatus = "completed"
	BulkStatusFailed     BulkOperationStatus = "failed"
)

type BulkOperationType string

const (
	BulkTypeUpdate BulkOperationType = "update"
	BulkTypeDelete BulkOperationType = "delete"
)

// BulkOperation represents a bulk update operation
type BulkOperation struct {
	ID             primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	UserID         primitive.ObjectID     `json:"user_id" bson:"user_id"`
	ModuleName     string                 `json:"module_name" bson:"module_name"`
	Type           BulkOperationType      `json:"type" bson:"type"` // "update" or "delete"
	Filters        map[string]interface{} `json:"filters" bson:"filters"`
	Updates        map[string]interface{} `json:"updates" bson:"updates"`
	Status         BulkOperationStatus    `json:"status" bson:"status"`
	TotalRecords   int                    `json:"total_records" bson:"total_records"`
	ProcessedCount int                    `json:"processed_count" bson:"processed_count"`
	SuccessCount   int                    `json:"success_count" bson:"success_count"`
	ErrorCount     int                    `json:"error_count" bson:"error_count"`
	Errors         []BulkError            `json:"errors,omitempty" bson:"errors,omitempty"`
	CreatedAt      time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" bson:"updated_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
}

// BulkError represents an error during bulk operation
type BulkError struct {
	RecordID string `json:"record_id" bson:"record_id"`
	Message  string `json:"message" bson:"message"`
}
