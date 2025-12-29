package service

import (
	"context"
	"errors"
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DashboardService interface {
	CreateDashboard(ctx context.Context, dashboard *models.DashboardConfig, userID primitive.ObjectID) error
	GetDashboard(ctx context.Context, id string, userID primitive.ObjectID) (*models.DashboardConfig, error)
	ListUserDashboards(ctx context.Context, userID primitive.ObjectID) ([]models.DashboardConfig, error)
	UpdateDashboard(ctx context.Context, id string, dashboard *models.DashboardConfig, userID primitive.ObjectID) error
	DeleteDashboard(ctx context.Context, id string, userID primitive.ObjectID) error
	SetDefaultDashboard(ctx context.Context, dashboardID string, userID primitive.ObjectID) error
	GetDashboardData(ctx context.Context, dashboardID string, userID primitive.ObjectID) (map[string]interface{}, error)
}

type DashboardServiceImpl struct {
	DashboardRepo repository.DashboardRepository
	RecordService RecordService
	ModuleRepo    repository.ModuleRepository
	ChartService  ChartService
}

func NewDashboardService(
	dashboardRepo repository.DashboardRepository,
	recordService RecordService,
	moduleRepo repository.ModuleRepository,
	chartService ChartService,
) DashboardService {
	return &DashboardServiceImpl{
		DashboardRepo: dashboardRepo,
		RecordService: recordService,
		ModuleRepo:    moduleRepo,
		ChartService:  chartService,
	}
}

func (s *DashboardServiceImpl) CreateDashboard(ctx context.Context, dashboard *models.DashboardConfig, userID primitive.ObjectID) error {
	dashboard.UserID = userID

	// Validate widget configurations
	if err := s.validateWidgets(ctx, dashboard.Widgets); err != nil {
		return err
	}

	return s.DashboardRepo.Create(ctx, dashboard)
}

func (s *DashboardServiceImpl) GetDashboard(ctx context.Context, id string, userID primitive.ObjectID) (*models.DashboardConfig, error) {
	dashboard, err := s.DashboardRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check permissions: user must own dashboard or it must be shared
	if dashboard.UserID != userID && !dashboard.IsShared {
		return nil, errors.New("access denied")
	}

	return dashboard, nil
}

func (s *DashboardServiceImpl) ListUserDashboards(ctx context.Context, userID primitive.ObjectID) ([]models.DashboardConfig, error) {
	return s.DashboardRepo.FindByUserID(ctx, userID.Hex())
}

func (s *DashboardServiceImpl) UpdateDashboard(ctx context.Context, id string, dashboard *models.DashboardConfig, userID primitive.ObjectID) error {
	// Get existing dashboard to check ownership
	existing, err := s.DashboardRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	if existing.UserID != userID {
		return errors.New("access denied: you can only update your own dashboards")
	}

	// Validate widget configurations
	if err := s.validateWidgets(ctx, dashboard.Widgets); err != nil {
		return err
	}

	return s.DashboardRepo.Update(ctx, id, dashboard)
}

func (s *DashboardServiceImpl) DeleteDashboard(ctx context.Context, id string, userID primitive.ObjectID) error {
	// Get existing dashboard to check ownership
	existing, err := s.DashboardRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	if existing.UserID != userID {
		return errors.New("access denied: you can only delete your own dashboards")
	}

	return s.DashboardRepo.Delete(ctx, id)
}

func (s *DashboardServiceImpl) SetDefaultDashboard(ctx context.Context, dashboardID string, userID primitive.ObjectID) error {
	return s.DashboardRepo.SetDefault(ctx, userID.Hex(), dashboardID)
}

func (s *DashboardServiceImpl) GetDashboardData(ctx context.Context, dashboardID string, userID primitive.ObjectID) (map[string]interface{}, error) {
	dashboard, err := s.GetDashboard(ctx, dashboardID, userID)
	if err != nil {
		return nil, err
	}

	widgetData := make(map[string]interface{})

	// Fetch data for each widget
	for _, widget := range dashboard.Widgets {
		data, err := s.getWidgetData(ctx, widget, userID)
		if err != nil {
			// Log error but continue with other widgets
			widgetData[widget.ID] = map[string]interface{}{
				"error": err.Error(),
			}
			continue
		}
		widgetData[widget.ID] = data
	}

	return widgetData, nil
}

func (s *DashboardServiceImpl) validateWidgets(ctx context.Context, widgets []models.DashboardWidget) error {
	for _, widget := range widgets {
		// Check if module exists
		if widget.ModuleName != "" {
			_, err := s.ModuleRepo.FindByName(ctx, widget.ModuleName)
			if err != nil {
				return fmt.Errorf("invalid module '%s' for widget '%s'", widget.ModuleName, widget.Title)
			}
		}

		// Validate widget type
		validTypes := map[string]bool{
			"metric": true,
			"chart":  true,
			"table":  true,
			"list":   true,
		}
		if !validTypes[widget.Type] {
			return fmt.Errorf("invalid widget type '%s'", widget.Type)
		}
	}
	return nil
}

func (s *DashboardServiceImpl) getWidgetData(ctx context.Context, widget models.DashboardWidget, userID primitive.ObjectID) (interface{}, error) {
	switch widget.Type {
	case "metric":
		return s.getMetricData(ctx, widget, userID)
	case "chart":
		return s.getChartData(ctx, widget, userID)
	case "table", "list":
		return s.getTableData(ctx, widget, userID)
	default:
		return nil, fmt.Errorf("unsupported widget type: %s", widget.Type)
	}
}

func (s *DashboardServiceImpl) getMetricData(ctx context.Context, widget models.DashboardWidget, userID primitive.ObjectID) (interface{}, error) {
	// Extract filters from widget config
	filters := make(map[string]interface{})
	if configFilters, ok := widget.Config["filters"].(map[string]interface{}); ok {
		filters = configFilters
	}

	// Get count of records
	_, count, err := s.RecordService.ListRecords(ctx, widget.ModuleName, filters, 1, 1, "created_at", "desc", userID)
	if err != nil {
		return nil, err
	}

	// Get aggregation value if specified
	aggregation := "count"
	if agg, ok := widget.Config["aggregation"].(string); ok {
		aggregation = agg
	}

	result := map[string]interface{}{
		"value":       count,
		"aggregation": aggregation,
	}

	// If aggregation is not just count, we need to calculate it
	if aggregation != "count" {
		field, ok := widget.Config["field"].(string)
		if !ok {
			return result, nil
		}

		// Fetch records to calculate aggregation
		records, _, err := s.RecordService.ListRecords(ctx, widget.ModuleName, filters, 1, 10000, "created_at", "desc", userID)
		if err != nil {
			return result, nil
		}

		value := s.calculateAggregation(records, field, aggregation)
		result["value"] = value
	}

	return result, nil
}

func (s *DashboardServiceImpl) getChartData(ctx context.Context, widget models.DashboardWidget, userID primitive.ObjectID) (interface{}, error) {
	// Extract chart config
	chartType, _ := widget.Config["chart_type"].(string)
	if chartType == "" {
		chartType = "bar"
	}

	filters := make(map[string]interface{})
	if configFilters, ok := widget.Config["filters"].(map[string]interface{}); ok {
		filters = configFilters
	}

	// Fetch records
	records, _, err := s.RecordService.ListRecords(ctx, widget.ModuleName, filters, 1, 1000, "created_at", "desc", userID)
	if err != nil {
		return nil, err
	}

	// Group by field if specified
	groupByField, _ := widget.Config["group_by"].(string)
	if groupByField == "" {
		groupByField = "status"
	}

	// Aggregate data
	chartData := s.groupRecords(records, groupByField)

	return map[string]interface{}{
		"type": chartType,
		"data": chartData,
	}, nil
}

func (s *DashboardServiceImpl) getTableData(ctx context.Context, widget models.DashboardWidget, userID primitive.ObjectID) (interface{}, error) {
	filters := make(map[string]interface{})
	if configFilters, ok := widget.Config["filters"].(map[string]interface{}); ok {
		filters = configFilters
	}

	limit := int64(5)
	if configLimit, ok := widget.Config["limit"].(float64); ok {
		limit = int64(configLimit)
	}

	records, total, err := s.RecordService.ListRecords(ctx, widget.ModuleName, filters, 1, limit, "created_at", "desc", userID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"records": records,
		"total":   total,
	}, nil
}

func (s *DashboardServiceImpl) calculateAggregation(records []map[string]any, field string, aggregation string) float64 {
	if len(records) == 0 {
		return 0
	}

	var sum float64
	var count int
	var min, max float64

	for i, record := range records {
		val, ok := record[field]
		if !ok {
			continue
		}

		var numVal float64
		switch v := val.(type) {
		case float64:
			numVal = v
		case int:
			numVal = float64(v)
		case int64:
			numVal = float64(v)
		default:
			continue
		}

		count++
		sum += numVal

		if i == 0 || numVal < min {
			min = numVal
		}
		if i == 0 || numVal > max {
			max = numVal
		}
	}

	switch aggregation {
	case "sum":
		return sum
	case "avg":
		if count > 0 {
			return sum / float64(count)
		}
		return 0
	case "min":
		return min
	case "max":
		return max
	default:
		return float64(count)
	}
}

func (s *DashboardServiceImpl) groupRecords(records []map[string]any, field string) []map[string]interface{} {
	groups := make(map[string]int)

	for _, record := range records {
		val, ok := record[field]
		if !ok {
			val = "Unknown"
		}

		key := fmt.Sprintf("%v", val)
		groups[key]++
	}

	var result []map[string]interface{}
	for key, count := range groups {
		result = append(result, map[string]interface{}{
			"name":  key,
			"value": count,
		})
	}

	return result
}
