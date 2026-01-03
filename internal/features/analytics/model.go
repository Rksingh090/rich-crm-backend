package analytics

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DataSourceType constants
const (
	DataSourceTypeCRM        = "crm"
	DataSourceTypeERP        = "erp"
	DataSourceTypePostgreSQL = "postgresql"
	DataSourceTypeMySQL      = "mysql"
	DataSourceTypeMongoDB    = "mongodb"
)

// DataSource represents a data source configuration
type DataSource struct {
	ID          primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	Name        string                 `json:"name" bson:"name"`
	Type        string                 `json:"type" bson:"type"` // "crm", "erp", "postgresql", "mysql", "mongodb"
	Description string                 `json:"description,omitempty" bson:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty" bson:"config,omitempty"`
	IsActive    bool                   `json:"is_active" bson:"is_active"`
	CreatedBy   primitive.ObjectID     `json:"created_by" bson:"created_by"`
	CreatedAt   time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" bson:"updated_at"`
}

// Metric represents a saved metric definition
type Metric struct {
	ID              primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	Name            string                 `json:"name" bson:"name"`
	Description     string                 `json:"description,omitempty" bson:"description,omitempty"`
	DataSourceID    string                 `json:"data_source_id" bson:"data_source_id"`     // Reference to DataSource ID (string to support external IDs if needed, but likely ObjectID hex)
	Module          string                 `json:"module" bson:"module"`                     // Table/Module name
	Field           string                 `json:"field" bson:"field"`                       // Field to aggregate
	AggregationType string                 `json:"aggregation_type" bson:"aggregation_type"` // "sum", "avg", "count", "min", "max"
	GroupBy         []string               `json:"group_by,omitempty" bson:"group_by,omitempty"`
	Filters         map[string]interface{} `json:"filters,omitempty" bson:"filters,omitempty"`
	CreatedBy       primitive.ObjectID     `json:"created_by,omitempty" bson:"created_by,omitempty"`
	CreatedAt       time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" bson:"updated_at"`
}

// MetricResult represents calculation result
type MetricResult struct {
	MetricID  string                   `json:"metric_id"`
	Value     interface{}              `json:"value"`          // Single value or array depending on grouping
	Data      []map[string]interface{} `json:"data,omitempty"` // Raw data if needed
	Timestamp time.Time                `json:"timestamp"`
}

// MetricDataPoint for history/charts
type MetricDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label,omitempty"`
}

// TimeRange for history queries
type TimeRange struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Interval string    `json:"interval"` // "day", "week", "month"
}
