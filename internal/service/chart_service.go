package service

import (
	"context"
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ChartService interface {
	CreateChart(ctx context.Context, chart *models.Chart) error
	GetChart(ctx context.Context, id string) (*models.Chart, error)
	ListCharts(ctx context.Context) ([]models.Chart, error)
	UpdateChart(ctx context.Context, id string, chart *models.Chart) error
	DeleteChart(ctx context.Context, id string) error
	GetChartData(ctx context.Context, id string) ([]map[string]interface{}, error)
}

type ChartServiceImpl struct {
	ChartRepo    repository.ChartRepository
	RecordRepo   repository.RecordRepository
	ModuleRepo   repository.ModuleRepository
	AuditService AuditService
}

func NewChartService(chartRepo repository.ChartRepository, recordRepo repository.RecordRepository, moduleRepo repository.ModuleRepository, auditService AuditService) ChartService {
	return &ChartServiceImpl{
		ChartRepo:    chartRepo,
		RecordRepo:   recordRepo,
		ModuleRepo:   moduleRepo,
		AuditService: auditService,
	}
}

func (s *ChartServiceImpl) CreateChart(ctx context.Context, chart *models.Chart) error {
	chart.CreatedAt = time.Now()
	chart.UpdatedAt = time.Now()
	chart.UpdatedAt = time.Now()
	err := s.ChartRepo.Create(ctx, chart)
	if err == nil {
		s.AuditService.LogChange(ctx, models.AuditActionChart, "charts", chart.ID.Hex(), map[string]models.Change{
			"chart": {New: chart},
		})
	}
	return err
}

func (s *ChartServiceImpl) GetChart(ctx context.Context, id string) (*models.Chart, error) {
	return s.ChartRepo.Get(ctx, id)
}

func (s *ChartServiceImpl) ListCharts(ctx context.Context) ([]models.Chart, error) {
	return s.ChartRepo.List(ctx)
}

func (s *ChartServiceImpl) UpdateChart(ctx context.Context, id string, chart *models.Chart) error {
	// Get old chart for audit
	oldChart, _ := s.GetChart(ctx, id)

	chart.UpdatedAt = time.Now()
	err := s.ChartRepo.Update(ctx, id, chart)
	if err == nil {
		s.AuditService.LogChange(ctx, models.AuditActionChart, "charts", id, map[string]models.Change{
			"chart": {Old: oldChart, New: chart},
		})
	}
	return err
}

func (s *ChartServiceImpl) DeleteChart(ctx context.Context, id string) error {
	// Get old chart for audit
	oldChart, _ := s.GetChart(ctx, id)

	err := s.ChartRepo.Delete(ctx, id)
	if err == nil {
		name := id
		if oldChart != nil {
			name = oldChart.Name
		}
		s.AuditService.LogChange(ctx, models.AuditActionChart, "charts", name, map[string]models.Change{
			"chart": {Old: oldChart, New: "DELETED"},
		})
	}
	return err
}

func (s *ChartServiceImpl) GetChartData(ctx context.Context, id string) ([]map[string]interface{}, error) {
	// 1. Get Chart Definition
	chart, err := s.ChartRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// 2. Identify Module
	// Assuming chart.ModuleID holds the module Name (e.g. "leads")
	moduleName := chart.ModuleID

	// 3. Construct Aggregation Pipeline
	pipeline := mongo.Pipeline{}

	// Grouping ID
	// If XAxisField is a Date, we might want to truncate.
	// We need Schema to know if it's a date.
	module, err := s.ModuleRepo.FindByName(ctx, moduleName)
	var groupID any = "$" + chart.XAxisField

	if err == nil {
		for _, f := range module.Fields {
			if f.Name == chart.XAxisField {
				if f.Type == models.FieldTypeDate {
					// Format Date to YYYY-MM-DD for grouping
					groupID = bson.M{
						"$dateToString": bson.M{
							"format": "%Y-%m-%d",
							"date":   "$" + chart.XAxisField,
						},
					}
				} else if f.Type == models.FieldTypeLookup {
					// Lookup field usually stores ObjectID.
					// We might want to group by the ObjectID string?
					// Or ideally getting the label would require $lookup aggregation which is complex efficiently.
					// For MVP, letting it group by ID is acceptable, frontend might see IDs.
				}
				break
			}
		}
	}

	// Accumulator
	var accumulator any
	switch chart.AggregationType {
	case models.AggregationTypeCount:
		accumulator = bson.M{"$sum": 1}
	case models.AggregationTypeSum:
		accumulator = bson.M{"$sum": "$" + chart.YAxisField}
	case models.AggregationTypeAvg:
		accumulator = bson.M{"$avg": "$" + chart.YAxisField}
	case models.AggregationTypeMin:
		accumulator = bson.M{"$min": "$" + chart.YAxisField}
	case models.AggregationTypeMax:
		accumulator = bson.M{"$max": "$" + chart.YAxisField}
	default:
		accumulator = bson.M{"$sum": 1}
	}

	groupStage := bson.D{
		{Key: "$group", Value: bson.D{
			{Key: "_id", Value: groupID},
			{Key: "value", Value: accumulator},
		}},
	}
	pipeline = append(pipeline, groupStage)

	// Sort by _id (X Axis)
	sortStage := bson.D{
		{Key: "$sort", Value: bson.D{
			{Key: "_id", Value: 1},
		}},
	}
	pipeline = append(pipeline, sortStage)

	// 4. Execute
	results, err := s.RecordRepo.Aggregate(ctx, moduleName, pipeline)
	if err != nil {
		return nil, err
	}

	// 5. Format Output
	// Flatten _id to "name"
	formatted := make([]map[string]interface{}, 0, len(results))
	for _, res := range results {
		name := "Unknown"
		if val, ok := res["_id"]; ok && val != nil {
			name = fmt.Sprintf("%v", val)
		}

		formatted = append(formatted, map[string]interface{}{
			"name":  name,
			"value": res["value"],
		})
	}

	return formatted, nil
}
