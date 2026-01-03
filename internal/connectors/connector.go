package connectors

import (
	"context"
	"time"
)

// DataSource represents a data source configuration
type DataSource struct {
	ID       string
	Name     string
	Type     string // "crm", "erp", "postgresql", "mysql", "mongodb"
	Config   map[string]interface{}
	IsActive bool
}

// QueryRequest represents a data query
type QueryRequest struct {
	Source      string                 // Data source ID
	Module      string                 // Module/table name
	Fields      []string               // Fields to retrieve
	Filters     map[string]interface{} // Filter conditions
	Sort        map[string]int         // Sort order (1 for ASC, -1 for DESC)
	Limit       int64
	Offset      int64
	Aggregation *AggregationConfig // Optional aggregation
}

// AggregationConfig for analytics queries
type AggregationConfig struct {
	GroupBy []string
	Metrics []MetricConfig
}

// MetricConfig defines a metric calculation
type MetricConfig struct {
	Field    string
	Function string // "sum", "avg", "count", "min", "max"
	Alias    string
}

// QueryResponse represents query results
type QueryResponse struct {
	Data       []map[string]interface{}
	TotalCount int64
	Timestamp  time.Time
}

// SchemaInfo represents module/table schema
type SchemaInfo struct {
	Module string
	Fields []FieldInfo
}

// FieldInfo represents field metadata
type FieldInfo struct {
	Name         string
	Type         string
	Label        string
	IsRequired   bool
	IsPrimaryKey bool
}

// Connector interface for all data sources
type Connector interface {
	// Connect establishes connection to data source
	Connect(ctx context.Context, config map[string]interface{}) error

	// Disconnect closes connection
	Disconnect(ctx context.Context) error

	// Query executes a query and returns results
	Query(ctx context.Context, req QueryRequest) (*QueryResponse, error)

	// GetSchema returns schema information for a module/table
	GetSchema(ctx context.Context, module string) (*SchemaInfo, error)

	// TestConnection tests if connection is valid
	TestConnection(ctx context.Context) error

	// GetType returns the connector type
	GetType() string
}
