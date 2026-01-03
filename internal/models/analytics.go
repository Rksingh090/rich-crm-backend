package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Metric represents a business metric definition
type Metric struct {
	ID              primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	Name            string                 `json:"name" bson:"name" validate:"required"`
	Description     string                 `json:"description" bson:"description"`
	DataSourceID    string                 `json:"data_source_id" bson:"data_source_id" validate:"required"`
	Module          string                 `json:"module" bson:"module" validate:"required"`
	Field           string                 `json:"field" bson:"field"`
	AggregationType string                 `json:"aggregation_type" bson:"aggregation_type" validate:"required,oneof=count sum avg min max"`
	GroupBy         []string               `json:"group_by" bson:"group_by"`
	Filters         map[string]interface{} `json:"filters" bson:"filters"`
	RefreshInterval int                    `json:"refresh_interval" bson:"refresh_interval"` // seconds, 0 = no auto-refresh
	CreatedBy       primitive.ObjectID     `json:"created_by" bson:"created_by"`
	CreatedAt       time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" bson:"updated_at"`
}

// MetricResult represents the result of a metric calculation
type MetricResult struct {
	MetricID  string                   `json:"metric_id"`
	Value     interface{}              `json:"value"`
	Data      []map[string]interface{} `json:"data"`
	Timestamp time.Time                `json:"timestamp"`
}

// MetricDataPoint represents a single data point in metric history
type MetricDataPoint struct {
	Timestamp time.Time   `json:"timestamp"`
	Value     interface{} `json:"value"`
}

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
