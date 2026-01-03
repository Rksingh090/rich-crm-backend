package connectors

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RecordProvider is an interface for accessing record data
type RecordProvider interface {
	ListRecords(ctx context.Context, moduleName string, filters map[string]any, limit int64, offset int64, sortField string, sortOrder string, userID primitive.ObjectID) ([]map[string]any, int64, error)
}

// ModuleProvider is an interface for accessing module schema
type ModuleProvider interface {
	GetModuleByName(ctx context.Context, name string, userID primitive.ObjectID) (Module, error)
}

// Module represents a simplified module structure
type Module struct {
	Name   string
	Fields []ModuleField
}

// ModuleField represents a simplified field structure
type ModuleField struct {
	Name     string
	Type     string
	Label    string
	Required bool
}

// CRMConnector provides access to internal CRM data
type CRMConnector struct {
	recordProvider RecordProvider
	moduleProvider ModuleProvider
	connectorType  string
}

// NewCRMConnector creates a new CRM connector
func NewCRMConnector(recordProvider RecordProvider, moduleProvider ModuleProvider) Connector {
	return &CRMConnector{
		recordProvider: recordProvider,
		moduleProvider: moduleProvider,
		connectorType:  "crm",
	}
}

// Connect is a no-op for CRM connector as it uses internal services
func (c *CRMConnector) Connect(ctx context.Context, config map[string]interface{}) error {
	return nil
}

// Disconnect is a no-op for CRM connector
func (c *CRMConnector) Disconnect(ctx context.Context) error {
	return nil
}

// Query executes a query against CRM data
func (c *CRMConnector) Query(ctx context.Context, req QueryRequest) (*QueryResponse, error) {
	// Get user ID from context (for permission checks)
	userID := primitive.NilObjectID
	if uid, ok := ctx.Value("user_id").(primitive.ObjectID); ok {
		userID = uid
	}

	// Prepare sort field and order
	sortField := ""
	sortOrder := ""
	if len(req.Sort) > 0 {
		for field, order := range req.Sort {
			sortField = field
			if order == -1 {
				sortOrder = "desc"
			} else {
				sortOrder = "asc"
			}
			break // Take first sort field
		}
	}

	// Fetch records using RecordProvider
	records, totalCount, err := c.recordProvider.ListRecords(
		ctx,
		req.Module,
		req.Filters,
		req.Limit,
		req.Offset,
		sortField,
		sortOrder,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query CRM data: %w", err)
	}

	// Apply field selection if specified
	if len(req.Fields) > 0 {
		records = c.selectFields(records, req.Fields)
	}

	// If aggregation is requested, perform it
	if req.Aggregation != nil {
		records = c.performAggregation(records, req.Aggregation)
		totalCount = int64(len(records))
	}

	return &QueryResponse{
		Data:       records,
		TotalCount: totalCount,
	}, nil
}

// GetSchema returns schema information for a CRM module
func (c *CRMConnector) GetSchema(ctx context.Context, module string) (*SchemaInfo, error) {
	userID := primitive.NilObjectID
	if uid, ok := ctx.Value("user_id").(primitive.ObjectID); ok {
		userID = uid
	}

	mod, err := c.moduleProvider.GetModuleByName(ctx, module, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get module schema: %w", err)
	}

	schema := &SchemaInfo{
		Module: module,
		Fields: make([]FieldInfo, 0, len(mod.Fields)),
	}

	for _, field := range mod.Fields {
		schema.Fields = append(schema.Fields, FieldInfo{
			Name:       field.Name,
			Type:       field.Type,
			Label:      field.Label,
			IsRequired: field.Required,
		})
	}

	return schema, nil
}

// TestConnection always returns nil for CRM connector
func (c *CRMConnector) TestConnection(ctx context.Context) error {
	return nil
}

// GetType returns the connector type
func (c *CRMConnector) GetType() string {
	return c.connectorType
}

// selectFields applies field selection to records
func (c *CRMConnector) selectFields(records []map[string]interface{}, fields []string) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(records))

	for _, record := range records {
		filteredRecord := make(map[string]interface{})
		for _, field := range fields {
			if val, ok := record[field]; ok {
				filteredRecord[field] = val
			}
		}
		result = append(result, filteredRecord)
	}

	return result
}

// performAggregation performs aggregation on records
func (c *CRMConnector) performAggregation(records []map[string]interface{}, agg *AggregationConfig) []map[string]interface{} {
	if len(agg.GroupBy) == 0 {
		// No grouping, just calculate metrics across all records
		result := make(map[string]interface{})
		for _, metric := range agg.Metrics {
			result[metric.Alias] = c.calculateMetric(records, metric)
		}
		return []map[string]interface{}{result}
	}

	// Group records
	groups := make(map[string][]map[string]interface{})
	for _, record := range records {
		key := c.buildGroupKey(record, agg.GroupBy)
		groups[key] = append(groups[key], record)
	}

	// Calculate metrics for each group
	result := make([]map[string]interface{}, 0, len(groups))
	for _, groupRecords := range groups {
		groupResult := make(map[string]interface{})

		// Add group by fields
		for i, field := range agg.GroupBy {
			if i == 0 && len(groupRecords) > 0 {
				groupResult[field] = groupRecords[0][field]
			}
		}

		// Calculate metrics
		for _, metric := range agg.Metrics {
			groupResult[metric.Alias] = c.calculateMetric(groupRecords, metric)
		}

		result = append(result, groupResult)
	}

	return result
}

// buildGroupKey creates a unique key for grouping
func (c *CRMConnector) buildGroupKey(record map[string]interface{}, fields []string) string {
	key := ""
	for _, field := range fields {
		if val, ok := record[field]; ok {
			key += fmt.Sprintf("%v|", val)
		}
	}
	return key
}

// calculateMetric calculates a metric value
func (c *CRMConnector) calculateMetric(records []map[string]interface{}, metric MetricConfig) interface{} {
	switch metric.Function {
	case "count":
		return len(records)
	case "sum":
		sum := 0.0
		for _, record := range records {
			if val, ok := record[metric.Field]; ok {
				if num, ok := val.(float64); ok {
					sum += num
				}
			}
		}
		return sum
	case "avg":
		sum := 0.0
		count := 0
		for _, record := range records {
			if val, ok := record[metric.Field]; ok {
				if num, ok := val.(float64); ok {
					sum += num
					count++
				}
			}
		}
		if count > 0 {
			return sum / float64(count)
		}
		return 0.0
	case "min":
		var min *float64
		for _, record := range records {
			if val, ok := record[metric.Field]; ok {
				if num, ok := val.(float64); ok {
					if min == nil || num < *min {
						min = &num
					}
				}
			}
		}
		if min != nil {
			return *min
		}
		return nil
	case "max":
		var max *float64
		for _, record := range records {
			if val, ok := record[metric.Field]; ok {
				if num, ok := val.(float64); ok {
					if max == nil || num > *max {
						max = &num
					}
				}
			}
		}
		if max != nil {
			return *max
		}
		return nil
	default:
		return nil
	}
}
