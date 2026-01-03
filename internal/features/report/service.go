package report

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/module"
	"go-crm/internal/features/record"

	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReportService interface {
	CreateReport(ctx context.Context, report *Report) error
	GetReport(ctx context.Context, id string) (*Report, error)
	ListReports(ctx context.Context) ([]Report, error)
	UpdateReport(ctx context.Context, id string, report *Report) error
	DeleteReport(ctx context.Context, id string) error
	RunReport(ctx context.Context, id string, userID primitive.ObjectID) ([]map[string]any, error)
	RunPivotReport(ctx context.Context, config *PivotConfig, moduleName string, filters map[string]any, userID primitive.ObjectID) (interface{}, error)
	RunCrossModuleReport(ctx context.Context, config *CrossModuleConfig, filters map[string]any, userID primitive.ObjectID) ([]map[string]any, error)
	ExportReport(ctx context.Context, id string, format string, userID primitive.ObjectID) ([]byte, string, error)
	ExportToExcel(ctx context.Context, data []map[string]any, columns []string, filename string) ([]byte, string, error)
}

type ReportServiceImpl struct {
	ReportRepo    ReportRepository
	RecordService record.RecordService
	ModuleService module.ModuleService
	AuditService  audit.AuditService
}

func NewReportService(reportRepo ReportRepository, recordService record.RecordService, moduleService module.ModuleService, auditService audit.AuditService) ReportService {
	return &ReportServiceImpl{
		ReportRepo:    reportRepo,
		RecordService: recordService,
		ModuleService: moduleService,
		AuditService:  auditService,
	}
}

func (s *ReportServiceImpl) CreateReport(ctx context.Context, report *Report) error {
	if report.ID.IsZero() {
		report.ID = primitive.NewObjectID()
	}
	err := s.ReportRepo.Create(ctx, report)
	if err == nil {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionReport, "reports", report.ID.Hex(), map[string]common_models.Change{
			"report": {New: report},
		})
	}
	return err
}

func (s *ReportServiceImpl) GetReport(ctx context.Context, id string) (*Report, error) {
	return s.ReportRepo.Get(ctx, id)
}

func (s *ReportServiceImpl) ListReports(ctx context.Context) ([]Report, error) {
	return s.ReportRepo.List(ctx)
}

func (s *ReportServiceImpl) UpdateReport(ctx context.Context, id string, report *Report) error {
	oldReport, _ := s.GetReport(ctx, id)
	err := s.ReportRepo.Update(ctx, id, report)
	if err == nil {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionReport, "reports", id, map[string]common_models.Change{
			"report": {Old: oldReport, New: report},
		})
	}
	return err
}

func (s *ReportServiceImpl) DeleteReport(ctx context.Context, id string) error {
	oldReport, _ := s.GetReport(ctx, id)
	err := s.ReportRepo.Delete(ctx, id)
	if err == nil {
		name := id
		if oldReport != nil {
			name = oldReport.Name
		}
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionReport, "reports", name, map[string]common_models.Change{
			"report": {Old: oldReport, New: "DELETED"},
		})
	}
	return err
}

func (s *ReportServiceImpl) RunReport(ctx context.Context, id string, userID primitive.ObjectID) ([]map[string]any, error) {
	report, err := s.GetReport(ctx, id)
	if err != nil {
		return nil, err
	}

	records, _, err := s.RecordService.ListRecords(ctx, report.ModuleID, s.convertFilters(report.Filters), 1, 10000, "created_at", "desc", userID)
	if err != nil {
		return nil, err
	}

	if len(report.Columns) > 0 {
		var filteredRecords []map[string]any
		for _, rec := range records {
			newRec := make(map[string]any)
			if val, ok := rec["_id"]; ok {
				newRec["_id"] = val
			}
			for _, col := range report.Columns {
				if val, ok := rec[col]; ok {
					newRec[col] = val
				}
			}
			filteredRecords = append(filteredRecords, newRec)
		}
		return filteredRecords, nil
	}

	return records, nil
}

func (s *ReportServiceImpl) ExportReport(ctx context.Context, id string, format string, userID primitive.ObjectID) ([]byte, string, error) {
	if format != "csv" {
		return nil, "", fmt.Errorf("unsupported format: %s", format)
	}

	report, err := s.GetReport(ctx, id)
	if err != nil {
		return nil, "", err
	}

	records, _, err := s.RecordService.ListRecords(ctx, report.ModuleID, s.convertFilters(report.Filters), 1, 100000, "created_at", "desc", userID)
	if err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := report.Columns
	if len(headers) == 0 {
		if len(records) > 0 {
			for k := range records[0] {
				headers = append(headers, k)
			}
		}
	}
	if err := writer.Write(headers); err != nil {
		return nil, "", err
	}

	for _, rec := range records {
		var row []string
		for _, col := range headers {
			val := rec[col]
			strVal := fmt.Sprintf("%v", val)
			if mapVal, ok := val.(map[string]interface{}); ok {
				if name, ok := mapVal["name"]; ok {
					strVal = fmt.Sprintf("%v", name)
				} else if originalName, ok := mapVal["original_filename"]; ok {
					strVal = fmt.Sprintf("%v", originalName)
				}
			} else if tVal, ok := val.(time.Time); ok {
				strVal = tVal.Format("2006-01-02 15:04:05")
			} else if oid, ok := val.(primitive.ObjectID); ok {
				strVal = oid.Hex()
			}
			row = append(row, strVal)
		}
		if err := writer.Write(row); err != nil {
			return nil, "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("%s_report_%s.csv", report.Name, time.Now().Format("20060102_150405"))
	return buf.Bytes(), filename, nil
}

func (s *ReportServiceImpl) RunPivotReport(ctx context.Context, config *PivotConfig, moduleName string, filters map[string]any, userID primitive.ObjectID) (interface{}, error) {
	records, _, err := s.RecordService.ListRecords(ctx, moduleName, s.convertFilters(filters), 1, 10000, "created_at", "desc", userID)
	if err != nil {
		return nil, err
	}

	pivotTable := make(map[string]map[string]interface{})
	rowValues := make(map[string]bool)
	colValues := make(map[string]bool)

	for _, record := range records {
		rowKey := s.buildKey(record, config.RowFields)
		colKey := s.buildKey(record, config.ColumnFields)
		rowValues[rowKey] = true
		colValues[colKey] = true
	}

	for _, record := range records {
		rowKey := s.buildKey(record, config.RowFields)
		colKey := s.buildKey(record, config.ColumnFields)

		if pivotTable[rowKey] == nil {
			pivotTable[rowKey] = make(map[string]interface{})
		}

		currentVal := pivotTable[rowKey][colKey]
		newVal := s.aggregateValue(currentVal, record, config.ValueField, config.Aggregation)
		pivotTable[rowKey][colKey] = newVal
	}

	result := map[string]interface{}{
		"rows":    convertToArray(rowValues),
		"columns": convertToArray(colValues),
		"data":    pivotTable,
	}

	return result, nil
}

func (s *ReportServiceImpl) RunCrossModuleReport(ctx context.Context, config *CrossModuleConfig, filters map[string]any, userID primitive.ObjectID) ([]map[string]any, error) {
	if config.BaseModule == "" {
		return nil, fmt.Errorf("base module is required")
	}

	baseRecords, _, err := s.RecordService.ListRecords(ctx, config.BaseModule, s.convertFilters(filters), 1, 10000, "created_at", "desc", userID)
	if err != nil {
		return nil, err
	}

	var result []map[string]any
	for _, baseRecord := range baseRecords {
		enrichedRecord := make(map[string]any)
		for k, v := range baseRecord {
			enrichedRecord[k] = v
		}

		for _, join := range config.Joins {
			lookupVal, exists := baseRecord[join.LookupField]
			if !exists {
				continue
			}

			var relatedID string
			switch v := lookupVal.(type) {
			case primitive.ObjectID:
				relatedID = v.Hex()
			case string:
				relatedID = v
			case map[string]interface{}:
				if id, ok := v["id"].(string); ok {
					relatedID = id
				}
			default:
				continue
			}

			if relatedID == "" {
				continue
			}

			relatedRecord, err := s.RecordService.GetRecord(ctx, join.ModuleName, relatedID, userID)
			if err != nil {
				continue
			}

			prefix := join.ModuleName + "_"
			if len(join.Fields) > 0 {
				for _, field := range join.Fields {
					if val, ok := relatedRecord[field]; ok {
						enrichedRecord[prefix+field] = val
					}
				}
			} else {
				for k, v := range relatedRecord {
					if k != "_id" {
						enrichedRecord[prefix+k] = v
					}
				}
			}
		}
		result = append(result, enrichedRecord)
	}

	return result, nil
}

func (s *ReportServiceImpl) ExportToExcel(ctx context.Context, data []map[string]any, columns []string, filename string) ([]byte, string, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Report"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, "", err
	}
	f.SetActiveSheet(index)

	if len(columns) == 0 && len(data) > 0 {
		for k := range data[0] {
			columns = append(columns, k)
		}
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})

	for i, col := range columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, col)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	for rowIdx, record := range data {
		for colIdx, col := range columns {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			val := record[col]
			switch v := val.(type) {
			case time.Time:
				f.SetCellValue(sheetName, cell, v.Format("2006-01-02 15:04:05"))
			case primitive.ObjectID:
				f.SetCellValue(sheetName, cell, v.Hex())
			case map[string]interface{}:
				if name, ok := v["name"]; ok {
					f.SetCellValue(sheetName, cell, fmt.Sprintf("%v", name))
				} else if fname, ok := v["original_filename"]; ok {
					f.SetCellValue(sheetName, cell, fmt.Sprintf("%v", fname))
				} else {
					f.SetCellValue(sheetName, cell, fmt.Sprintf("%v", v))
				}
			default:
				f.SetCellValue(sheetName, cell, v)
			}
		}
	}

	for i := range columns {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheetName, col, col, 15)
	}

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}

	xlsxFilename := filename
	if !strings.HasSuffix(xlsxFilename, ".xlsx") {
		xlsxFilename += ".xlsx"
	}

	return buffer.Bytes(), xlsxFilename, nil
}

func (s *ReportServiceImpl) buildKey(record map[string]any, fields []string) string {
	var parts []string
	for _, field := range fields {
		if val, ok := record[field]; ok {
			parts = append(parts, fmt.Sprintf("%v", val))
		} else {
			parts = append(parts, "N/A")
		}
	}
	return strings.Join(parts, "|")
}

func (s *ReportServiceImpl) aggregateValue(current interface{}, record map[string]any, valueField string, aggregation string) interface{} {
	if aggregation == "count" {
		if current == nil {
			return 1
		}
		return current.(int) + 1
	}

	val, exists := record[valueField]
	if !exists {
		return current
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
		return current
	}

	if current == nil {
		return numVal
	}

	currentNum := current.(float64)
	switch aggregation {
	case "sum", "avg":
		return currentNum + numVal
	case "min":
		if numVal < currentNum {
			return numVal
		}
		return currentNum
	case "max":
		if numVal > currentNum {
			return numVal
		}
		return currentNum
	default:
		return current
	}
}

func convertToArray(m map[string]bool) []string {
	var result []string
	for k := range m {
		result = append(result, k)
	}
	return result
}

func (s *ReportServiceImpl) convertFilters(filters map[string]any) []common_models.Filter {
	var filterSlice []common_models.Filter
	for k, v := range filters {
		fieldName := k
		operator := "eq"
		if strings.Contains(k, "__") {
			parts := strings.Split(k, "__")
			if len(parts) == 2 {
				fieldName = parts[0]
				operator = parts[1]
			}
		}
		filterSlice = append(filterSlice, common_models.Filter{
			Field:    fieldName,
			Operator: operator,
			Value:    v,
		})
	}
	return filterSlice
}
