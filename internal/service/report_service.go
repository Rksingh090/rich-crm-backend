package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReportService interface {
	CreateReport(ctx context.Context, report *models.Report) error
	GetReport(ctx context.Context, id string) (*models.Report, error)
	ListReports(ctx context.Context) ([]models.Report, error)
	UpdateReport(ctx context.Context, id string, report *models.Report) error
	DeleteReport(ctx context.Context, id string) error
	RunReport(ctx context.Context, id string, userID primitive.ObjectID) ([]map[string]any, error)
	RunPivotReport(ctx context.Context, config *models.PivotConfig, moduleName string, filters map[string]any, userID primitive.ObjectID) (interface{}, error)
	RunCrossModuleReport(ctx context.Context, config *models.CrossModuleConfig, filters map[string]any, userID primitive.ObjectID) ([]map[string]any, error)
	ExportReport(ctx context.Context, id string, format string, userID primitive.ObjectID) ([]byte, string, error)
	ExportToExcel(ctx context.Context, data []map[string]any, columns []string, filename string) ([]byte, string, error)
}

type ReportServiceImpl struct {
	ReportRepo    repository.ReportRepository
	RecordService RecordService
	ModuleService ModuleService
}

func NewReportService(reportRepo repository.ReportRepository, recordService RecordService, moduleService ModuleService) ReportService {
	return &ReportServiceImpl{
		ReportRepo:    reportRepo,
		RecordService: recordService,
		ModuleService: moduleService,
	}
}

func (s *ReportServiceImpl) CreateReport(ctx context.Context, report *models.Report) error {
	if report.ID.IsZero() {
		report.ID = primitive.NewObjectID()
	}
	return s.ReportRepo.Create(ctx, report)
}

func (s *ReportServiceImpl) GetReport(ctx context.Context, id string) (*models.Report, error) {
	return s.ReportRepo.Get(ctx, id)
}

func (s *ReportServiceImpl) ListReports(ctx context.Context) ([]models.Report, error) {
	return s.ReportRepo.List(ctx)
}

func (s *ReportServiceImpl) UpdateReport(ctx context.Context, id string, report *models.Report) error {
	return s.ReportRepo.Update(ctx, id, report)
}

func (s *ReportServiceImpl) DeleteReport(ctx context.Context, id string) error {
	return s.ReportRepo.Delete(ctx, id)
}

func (s *ReportServiceImpl) RunReport(ctx context.Context, id string, userID primitive.ObjectID) ([]map[string]any, error) {
	// 1. Fetch Report Definition
	report, err := s.GetReport(ctx, id)
	if err != nil {
		return nil, err
	}

	// 2. Fetch Module to Verify Module Name
	// In Report model, ModuleID is likely the module's Name (string identifier) based on usage
	// Let's assume ModuleID refers to Module.Name for now, or fetch by ID if strictly ID.
	// implementation_plan.md said "ModuleID (string) - The source module". RecordService uses Name.
	// Let's assume it maps to "leads", "contacts" etc.
	moduleName := report.ModuleID

	// 3. Fetch Records with Filters
	// We want ALL records for report, so high limit? Or pagination?
	// For "Run", maybe limit 100 for preview. For "Export", unlimited.
	// Let's assume RunReport fetches all for now (be careful with large datasets).
	// A better approach for UI is RunReport returns paginated, but for simplicity here:
	// A better approach for UI is RunReport returns paginated, but for simplicity here:
	// A better approach for UI is RunReport returns paginated, but for simplicity here:
	records, _, err := s.RecordService.ListRecords(ctx, moduleName, report.Filters, 1, 10000, "created_at", "desc", userID)
	if err != nil {
		return nil, err
	}

	// 4. Filter Columns
	// If Columns are specified, filter the map keys.
	// Also flatten basic types if needed for display?
	if len(report.Columns) > 0 {
		var filteredRecords []map[string]any
		for _, rec := range records {
			newRec := make(map[string]any)
			// Always include ID ?
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

	// Fetch Data (All)
	records, _, err := s.RecordService.ListRecords(ctx, report.ModuleID, report.Filters, 1, 100000, "created_at", "desc", userID)
	if err != nil {
		return nil, "", err
	}

	// Generate CSV
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Headers
	// If columns defined, use them. Else fetch from module schema?
	headers := report.Columns
	if len(headers) == 0 {
		// Fallback: collect keys from first record or fetch schema
		// Ideally fetch schema
		// For simplicity, just use keys from first record if exists
		if len(records) > 0 {
			for k := range records[0] {
				headers = append(headers, k)
			}
		}
	}
	if err := writer.Write(headers); err != nil {
		return nil, "", err
	}

	// Rows
	for _, rec := range records {
		var row []string
		for _, col := range headers {
			val := rec[col]
			strVal := fmt.Sprintf("%v", val)
			// Handle complex types (objects) gracefully
			if mapVal, ok := val.(map[string]interface{}); ok {
				// e.g. Lookup or File object
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

// RunPivotReport executes a pivot table report
func (s *ReportServiceImpl) RunPivotReport(ctx context.Context, config *models.PivotConfig, moduleName string, filters map[string]any, userID primitive.ObjectID) (interface{}, error) {
	// Fetch all records
	records, _, err := s.RecordService.ListRecords(ctx, moduleName, filters, 1, 10000, "created_at", "desc", userID)
	if err != nil {
		return nil, err
	}

	// Build pivot table structure
	pivotTable := make(map[string]map[string]interface{})
	rowValues := make(map[string]bool)
	colValues := make(map[string]bool)

	// First pass: collect unique row and column values
	for _, record := range records {
		rowKey := s.buildKey(record, config.RowFields)
		colKey := s.buildKey(record, config.ColumnFields)
		rowValues[rowKey] = true
		colValues[colKey] = true
	}

	// Second pass: calculate aggregations
	for _, record := range records {
		rowKey := s.buildKey(record, config.RowFields)
		colKey := s.buildKey(record, config.ColumnFields)

		if pivotTable[rowKey] == nil {
			pivotTable[rowKey] = make(map[string]interface{})
		}

		// Apply aggregation
		currentVal := pivotTable[rowKey][colKey]
		newVal := s.aggregateValue(currentVal, record, config.ValueField, config.Aggregation)
		pivotTable[rowKey][colKey] = newVal
	}

	// Convert to array format for frontend
	result := map[string]interface{}{
		"rows":    convertToArray(rowValues),
		"columns": convertToArray(colValues),
		"data":    pivotTable,
	}

	return result, nil
}

// RunCrossModuleReport executes a cross-module report with joins
func (s *ReportServiceImpl) RunCrossModuleReport(ctx context.Context, config *models.CrossModuleConfig, filters map[string]any, userID primitive.ObjectID) ([]map[string]any, error) {
	if config.BaseModule == "" {
		return nil, fmt.Errorf("base module is required")
	}

	// Fetch base module records
	baseRecords, _, err := s.RecordService.ListRecords(ctx, config.BaseModule, filters, 1, 10000, "created_at", "desc", userID)
	if err != nil {
		return nil, err
	}

	// For each record, join with related modules
	var result []map[string]any
	for _, baseRecord := range baseRecords {
		enrichedRecord := make(map[string]any)

		// Copy base record fields
		for k, v := range baseRecord {
			enrichedRecord[k] = v
		}

		// Process joins
		for _, join := range config.Joins {
			// Get the lookup field value (ID of related record)
			lookupVal, exists := baseRecord[join.LookupField]
			if !exists {
				continue
			}

			// Convert to string ID
			var relatedID string
			switch v := lookupVal.(type) {
			case primitive.ObjectID:
				relatedID = v.Hex()
			case string:
				relatedID = v
			case map[string]interface{}:
				// Already populated lookup
				if id, ok := v["id"].(string); ok {
					relatedID = id
				}
			default:
				continue
			}

			if relatedID == "" {
				continue
			}

			// Fetch related record
			relatedRecord, err := s.RecordService.GetRecord(ctx, join.ModuleName, relatedID, userID)
			if err != nil {
				continue
			}

			// Add specified fields from related module
			prefix := join.ModuleName + "_"
			if len(join.Fields) > 0 {
				for _, field := range join.Fields {
					if val, ok := relatedRecord[field]; ok {
						enrichedRecord[prefix+field] = val
					}
				}
			} else {
				// If no fields specified, add all
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

// ExportToExcel exports data to Excel format
func (s *ReportServiceImpl) ExportToExcel(ctx context.Context, data []map[string]any, columns []string, filename string) ([]byte, string, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Report"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, "", err
	}

	// Set active sheet
	f.SetActiveSheet(index)

	// Write headers
	if len(columns) == 0 && len(data) > 0 {
		// If no columns specified, use keys from first record
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

	// Write data rows
	for rowIdx, record := range data {
		for colIdx, col := range columns {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			val := record[col]

			// Handle different types
			switch v := val.(type) {
			case time.Time:
				f.SetCellValue(sheetName, cell, v.Format("2006-01-02 15:04:05"))
			case primitive.ObjectID:
				f.SetCellValue(sheetName, cell, v.Hex())
			case map[string]interface{}:
				// For lookup or file objects, show name
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

	// Auto-fit columns
	for i := range columns {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheetName, col, col, 15)
	}

	// Save to buffer
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

// Helper functions
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
