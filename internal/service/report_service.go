package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReportService interface {
	CreateReport(ctx context.Context, report *models.Report) error
	GetReport(ctx context.Context, id string) (*models.Report, error)
	ListReports(ctx context.Context) ([]models.Report, error)
	UpdateReport(ctx context.Context, id string, report *models.Report) error
	DeleteReport(ctx context.Context, id string) error
	RunReport(ctx context.Context, id string, userID primitive.ObjectID) ([]map[string]any, error)
	ExportReport(ctx context.Context, id string, format string, userID primitive.ObjectID) ([]byte, string, error)
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
	// Fetch Data (All)
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
