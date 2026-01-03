package report

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReportType string

const (
	ReportTypeStandard    ReportType = "standard"
	ReportTypePivot       ReportType = "pivot"
	ReportTypeCrossModule ReportType = "cross_module"
)

// PivotConfig defines pivot table configuration
type PivotConfig struct {
	RowFields    []string `json:"row_fields" bson:"row_fields"`       // Fields to group by for rows
	ColumnFields []string `json:"column_fields" bson:"column_fields"` // Fields to group by for columns
	ValueField   string   `json:"value_field" bson:"value_field"`     // Field to aggregate
	Aggregation  string   `json:"aggregation" bson:"aggregation"`     // count, sum, avg, min, max
}

// ModuleJoin defines how to join related modules
type ModuleJoin struct {
	ModuleName  string   `json:"module_name" bson:"module_name"`   // Related module name
	LookupField string   `json:"lookup_field" bson:"lookup_field"` // Field in base module that references this module
	Fields      []string `json:"fields" bson:"fields"`             // Fields to include from this module
}

// CrossModuleConfig defines cross-module report configuration
type CrossModuleConfig struct {
	BaseModule string       `json:"base_module" bson:"base_module"` // Primary module
	Joins      []ModuleJoin `json:"joins" bson:"joins"`             // Related modules to join
}

// Report represents a saved report configuration
type Report struct {
	ID                primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name              string             `json:"name" bson:"name"`
	Description       string             `json:"description" bson:"description"`
	ReportType        ReportType         `json:"report_type" bson:"report_type"` // standard, pivot, cross_module
	ModuleID          string             `json:"module_id" bson:"module_id"`     // The module this report is based on
	Columns           []string           `json:"columns" bson:"columns"`         // List of field names to display
	Filters           map[string]any     `json:"filters" bson:"filters"`         // Stored query filters
	PivotConfig       *PivotConfig       `json:"pivot_config,omitempty" bson:"pivot_config,omitempty"`
	CrossModuleConfig *CrossModuleConfig `json:"cross_module_config,omitempty" bson:"cross_module_config,omitempty"`
	ChartType         string             `json:"chart_type,omitempty" bson:"chart_type,omitempty"` // bar, line, pie, table
	CreatedBy         primitive.ObjectID `json:"created_by,omitempty" bson:"created_by,omitempty"`
	UpdatedBy         primitive.ObjectID `json:"updated_by,omitempty" bson:"updated_by,omitempty"`
	CreatedAt         time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at" bson:"updated_at"`
}
