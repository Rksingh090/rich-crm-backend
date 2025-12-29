package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ImportStatus string

const (
	ImportStatusPending    ImportStatus = "pending"
	ImportStatusProcessing ImportStatus = "processing"
	ImportStatusCompleted  ImportStatus = "completed"
	ImportStatusFailed     ImportStatus = "failed"
)

// ImportJob represents a data import job
type ImportJob struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID           primitive.ObjectID `json:"user_id" bson:"user_id"`
	ModuleName       string             `json:"module_name" bson:"module_name"`
	FileName         string             `json:"file_name" bson:"file_name"`
	FilePath         string             `json:"file_path" bson:"file_path"`
	Status           ImportStatus       `json:"status" bson:"status"`
	TotalRecords     int                `json:"total_records" bson:"total_records"`
	ProcessedRecords int                `json:"processed_records" bson:"processed_records"`
	SuccessCount     int                `json:"success_count" bson:"success_count"`
	ErrorCount       int                `json:"error_count" bson:"error_count"`
	ColumnMapping    map[string]string  `json:"column_mapping" bson:"column_mapping"` // CSV column -> field mapping
	Errors           []ImportError      `json:"errors,omitempty" bson:"errors,omitempty"`
	CreatedAt        time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at" bson:"updated_at"`
	CompletedAt      *time.Time         `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
}

// ImportError represents an error during import
type ImportError struct {
	Row     int    `json:"row" bson:"row"`
	Field   string `json:"field" bson:"field"`
	Message string `json:"message" bson:"message"`
}

// ImportPreview represents a preview of import data
type ImportPreview struct {
	Headers      []string                 `json:"headers"`
	SampleData   []map[string]interface{} `json:"sample_data"`
	TotalRows    int                      `json:"total_rows"`
	ModuleFields []ModuleField            `json:"module_fields"`
}
