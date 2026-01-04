package chart

import (
	"context"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/module"
	"go-crm/internal/features/record"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ChartService interface {
	CreateChart(ctx context.Context, chart *Chart) error
	GetChart(ctx context.Context, id string) (*Chart, error)
	ListCharts(ctx context.Context) ([]Chart, error)
	UpdateChart(ctx context.Context, id string, chart *Chart) error
	DeleteChart(ctx context.Context, id string) error
	GetChartData(ctx context.Context, id string) ([]map[string]interface{}, error)
}

type ChartServiceImpl struct {
	ChartRepo    ChartRepository
	RecordRepo   record.RecordRepository
	ModuleRepo   module.ModuleRepository
	AuditService audit.AuditService
}

func NewChartService(chartRepo ChartRepository, recordRepo record.RecordRepository, moduleRepo module.ModuleRepository, auditService audit.AuditService) ChartService {
	return &ChartServiceImpl{
		ChartRepo:    chartRepo,
		RecordRepo:   recordRepo,
		ModuleRepo:   moduleRepo,
		AuditService: auditService,
	}
}

func (s *ChartServiceImpl) CreateChart(ctx context.Context, chart *Chart) error {
	chart.CreatedAt = time.Now()
	chart.UpdatedAt = time.Now()
	err := s.ChartRepo.Create(ctx, chart)
	if err == nil {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionChart, "charts", chart.ID.Hex(), map[string]common_models.Change{
			"chart": {New: chart},
		})
	}
	return err
}

func (s *ChartServiceImpl) GetChart(ctx context.Context, id string) (*Chart, error) {
	return s.ChartRepo.Get(ctx, id)
}

func (s *ChartServiceImpl) ListCharts(ctx context.Context) ([]Chart, error) {
	return s.ChartRepo.List(ctx)
}

func (s *ChartServiceImpl) UpdateChart(ctx context.Context, id string, chart *Chart) error {
	oldChart, _ := s.GetChart(ctx, id)

	chart.UpdatedAt = time.Now()
	err := s.ChartRepo.Update(ctx, id, chart)
	if err == nil {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionChart, "charts", id, map[string]common_models.Change{
			"chart": {Old: oldChart, New: chart},
		})
	}
	return err
}

func (s *ChartServiceImpl) DeleteChart(ctx context.Context, id string) error {
	oldChart, _ := s.GetChart(ctx, id)

	err := s.ChartRepo.Delete(ctx, id)
	if err == nil {
		name := id
		if oldChart != nil {
			name = oldChart.Name
		}
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionChart, "charts", name, map[string]common_models.Change{
			"chart": {Old: oldChart, New: "DELETED"},
		})
	}
	return err
}

func (s *ChartServiceImpl) GetChartData(ctx context.Context, id string) ([]map[string]interface{}, error) {
	chart, err := s.ChartRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	moduleName := chart.ModuleID
	pipeline := mongo.Pipeline{}

	mod, err := s.ModuleRepo.FindByName(ctx, moduleName)
	var groupID any = "$" + chart.XAxisField

	if err == nil {
		for _, f := range mod.Fields {
			if f.Name == chart.XAxisField {
				if f.Type == common_models.FieldTypeDate {
					groupID = bson.M{
						"$dateToString": bson.M{
							"format": "%Y-%m-%d",
							"date":   "$" + chart.XAxisField,
						},
					}
				}
				break
			}
		}
	}

	var accumulator any
	switch chart.AggregationType {
	case AggregationTypeCount:
		accumulator = bson.M{"$sum": 1}
	case AggregationTypeSum:
		accumulator = bson.M{"$sum": "$" + chart.YAxisField}
	case AggregationTypeAvg:
		accumulator = bson.M{"$avg": "$" + chart.YAxisField}
	case AggregationTypeMin:
		accumulator = bson.M{"$min": "$" + chart.YAxisField}
	case AggregationTypeMax:
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

	sortDir := 1
	sortField := "_id"

	if chart.ChartType == ChartTypeFunnel {
		sortField = "value"
		sortDir = -1
	}

	sortStage := bson.D{
		{Key: "$sort", Value: bson.D{
			{Key: sortField, Value: sortDir},
		}},
	}
	pipeline = append(pipeline, sortStage)

	results, err := s.RecordRepo.Aggregate(ctx, moduleName, pipeline)
	if err != nil {
		return nil, err
	}

	var selectOptions []common_models.SelectOptions
	if mod != nil {
		for _, f := range mod.Fields {
			if f.Name == chart.XAxisField && f.Type == common_models.FieldTypeSelect {
				selectOptions = f.Options
				break
			}
		}
	}

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

	if len(selectOptions) > 0 {
		optionMap := make(map[string]int)
		for i, opt := range selectOptions {
			optionMap[opt.Value] = i
		}

		sort.Slice(formatted, func(i, j int) bool {
			idxI, okI := optionMap[fmt.Sprintf("%v", formatted[i]["name"])]
			idxJ, okJ := optionMap[fmt.Sprintf("%v", formatted[j]["name"])]
			if !okI {
				idxI = 999
			}
			if !okJ {
				idxJ = 999
			}
			return idxI < idxJ
		})
	}

	return formatted, nil
}
