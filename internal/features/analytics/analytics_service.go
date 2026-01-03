package analytics

import (
	"context"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/connectors"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/chart"
	"go-crm/internal/features/dashboard"
	"go-crm/pkg/utils"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AnalyticsService interface {
	// Metric Management
	CreateMetric(ctx context.Context, metric *Metric) error
	GetMetric(ctx context.Context, id string) (*Metric, error)
	ListMetrics(ctx context.Context) ([]Metric, error)
	UpdateMetric(ctx context.Context, id string, updates map[string]interface{}) error
	DeleteMetric(ctx context.Context, id string) error

	// Metric Execution
	CalculateMetric(ctx context.Context, id string, filters map[string]interface{}) (*MetricResult, error)
	GetMetricHistory(ctx context.Context, id string, timeRange TimeRange) ([]MetricDataPoint, error)

	// Dashboard Analytics
	GetDashboardMetrics(ctx context.Context, dashboardID string) (map[string]*MetricResult, error)
}

type AnalyticsServiceImpl struct {
	metricRepo        MetricRepository
	dataSourceService DataSourceService
	chartService      chart.ChartService
	dashboardService  dashboard.DashboardService
	auditService      audit.AuditService
}

func NewAnalyticsService(
	metricRepo MetricRepository,
	dataSourceService DataSourceService,
	chartService chart.ChartService,
	dashboardService dashboard.DashboardService,
	auditService audit.AuditService,
) AnalyticsService {
	return &AnalyticsServiceImpl{
		metricRepo:        metricRepo,
		dataSourceService: dataSourceService,
		chartService:      chartService,
		dashboardService:  dashboardService,
		auditService:      auditService,
	}
}

func (s *AnalyticsServiceImpl) CreateMetric(ctx context.Context, metric *Metric) error {
	metric.CreatedAt = time.Now()
	metric.UpdatedAt = time.Now()

	err := s.metricRepo.Create(ctx, metric)
	if err == nil {
		s.auditService.LogChange(ctx, common_models.AuditActionCreate, "metrics", metric.ID.Hex(), map[string]common_models.Change{
			"metric": {New: metric},
		})
	}
	return err
}

func (s *AnalyticsServiceImpl) GetMetric(ctx context.Context, id string) (*Metric, error) {
	return s.metricRepo.Get(ctx, id)
}

func (s *AnalyticsServiceImpl) ListMetrics(ctx context.Context) ([]Metric, error) {
	return s.metricRepo.List(ctx)
}

func (s *AnalyticsServiceImpl) UpdateMetric(ctx context.Context, id string, updates map[string]interface{}) error {
	oldMetric, _ := s.GetMetric(ctx, id)

	updates["updated_at"] = time.Now()
	err := s.metricRepo.Update(ctx, id, updates)

	if err == nil {
		s.auditService.LogChange(ctx, common_models.AuditActionUpdate, "metrics", id, map[string]common_models.Change{
			"metric": {Old: oldMetric, New: updates},
		})
	}
	return err
}

func (s *AnalyticsServiceImpl) DeleteMetric(ctx context.Context, id string) error {
	oldMetric, _ := s.GetMetric(ctx, id)

	err := s.metricRepo.Delete(ctx, id)
	if err == nil {
		s.auditService.LogChange(ctx, common_models.AuditActionDelete, "metrics", id, map[string]common_models.Change{
			"metric": {Old: oldMetric, New: "DELETED"},
		})
	}
	return err
}

func (s *AnalyticsServiceImpl) CalculateMetric(ctx context.Context, id string, filters map[string]interface{}) (*MetricResult, error) {
	metric, err := s.GetMetric(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("metric not found: %w", err)
	}

	// Merge metric filters with provided filters
	mergedFilters := make(map[string]interface{})
	for k, v := range metric.Filters {
		mergedFilters[k] = v
	}
	for k, v := range filters {
		mergedFilters[k] = v
	}

	// Build query from metric definition
	query := connectors.QueryRequest{
		Source:  metric.DataSourceID,
		Module:  metric.Module,
		Filters: mergedFilters,
	}

	// Add aggregation if needed
	if metric.AggregationType != "" {
		query.Aggregation = &connectors.AggregationConfig{
			GroupBy: metric.GroupBy,
			Metrics: []connectors.MetricConfig{
				{
					Field:    metric.Field,
					Function: metric.AggregationType,
					Alias:    "value",
				},
			},
		}
	}

	// Execute query
	response, err := s.dataSourceService.QueryDataSource(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute metric query: %w", err)
	}

	// Process results
	return s.processMetricResults(metric, response), nil
}

func (s *AnalyticsServiceImpl) GetMetricHistory(ctx context.Context, id string, timeRange TimeRange) ([]MetricDataPoint, error) {
	// This would typically query a time-series database or cache
	// For now, return empty array as placeholder
	return []MetricDataPoint{}, nil
}

func (s *AnalyticsServiceImpl) GetDashboardMetrics(ctx context.Context, dashboardID string) (map[string]*MetricResult, error) {
	// Extract user ID from context
	if claims, ok := ctx.Value(utils.UserClaimsKey).(*utils.UserClaims); ok {
		userID, err := primitive.ObjectIDFromHex(claims.UserID)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID in context: %w", err)
		}
		dashboard, err := s.dashboardService.GetDashboard(ctx, dashboardID, userID)
		if err != nil {
			return nil, fmt.Errorf("dashboard not found: %w", err)
		}
		return s.processDashboardMetrics(ctx, dashboard)
	}

	return nil, fmt.Errorf("user not authenticated")
}

// processDashboardMetrics processes dashboard widgets and returns metric results
func (s *AnalyticsServiceImpl) processDashboardMetrics(ctx context.Context, dashboard *dashboard.DashboardConfig) (map[string]*MetricResult, error) {
	results := make(map[string]*MetricResult)

	// Get metrics for each widget in the dashboard
	for _, widget := range dashboard.Widgets {
		if widget.ID != "" {
			// Get chart data (charts can be treated as metrics)
			chartData, err := s.chartService.GetChartData(ctx, widget.ID)
			if err != nil {
				continue
			}

			results[widget.ID] = &MetricResult{
				MetricID:  widget.ID,
				Data:      chartData,
				Timestamp: time.Now(),
			}
		}
	}

	return results, nil
}

// processMetricResults processes query results into metric result format
func (s *AnalyticsServiceImpl) processMetricResults(metric *Metric, response *connectors.QueryResponse) *MetricResult {
	result := &MetricResult{
		MetricID:  metric.ID.Hex(),
		Data:      response.Data,
		Timestamp: time.Now(),
	}

	// If there's a single aggregated value, extract it
	if len(response.Data) == 1 && len(metric.GroupBy) == 0 {
		if val, ok := response.Data[0]["value"]; ok {
			result.Value = val
		}
	} else if len(response.Data) > 0 {
		// For grouped results, the value is the data array itself
		result.Value = response.Data
	}

	return result
}
