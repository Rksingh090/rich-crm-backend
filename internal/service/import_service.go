package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"io"
	"os"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ImportService interface {
	CreateJob(ctx context.Context, job *models.ImportJob) error
	GetJob(ctx context.Context, id string) (*models.ImportJob, error)
	GetUserJobs(ctx context.Context, userID primitive.ObjectID) ([]models.ImportJob, error)
	PreviewFile(ctx context.Context, file io.Reader, filename string, moduleName string) (*models.ImportPreview, error)
	ProcessImport(ctx context.Context, jobID string, userID primitive.ObjectID) error
	ProcessImportWithData(ctx context.Context, data []map[string]interface{}, columnMapping map[string]string, moduleName string, userID primitive.ObjectID, jobID string) error
}

type ImportServiceImpl struct {
	ImportRepo    repository.ImportRepository
	RecordService RecordService
	ModuleService ModuleService
}

func NewImportService(
	importRepo repository.ImportRepository,
	recordService RecordService,
	moduleService ModuleService,
) ImportService {
	return &ImportServiceImpl{
		ImportRepo:    importRepo,
		RecordService: recordService,
		ModuleService: moduleService,
	}
}

func (s *ImportServiceImpl) CreateJob(ctx context.Context, job *models.ImportJob) error {
	return s.ImportRepo.Create(ctx, job)
}

func (s *ImportServiceImpl) GetJob(ctx context.Context, id string) (*models.ImportJob, error) {
	return s.ImportRepo.Get(ctx, id)
}

func (s *ImportServiceImpl) GetUserJobs(ctx context.Context, userID primitive.ObjectID) ([]models.ImportJob, error) {
	return s.ImportRepo.FindByUserID(ctx, userID.Hex(), 50)
}

func (s *ImportServiceImpl) PreviewFile(ctx context.Context, file io.Reader, filename string, moduleName string) (*models.ImportPreview, error) {
	// Get module fields - use empty ObjectID for system access
	module, err := s.ModuleService.GetModuleByName(ctx, moduleName, primitive.NilObjectID)
	if err != nil {
		return nil, fmt.Errorf("module not found: %w", err)
	}

	var headers []string
	var sampleData []map[string]interface{}
	var totalRows int

	if strings.HasSuffix(strings.ToLower(filename), ".csv") {
		headers, sampleData, totalRows, err = s.parseCSV(file)
	} else if strings.HasSuffix(strings.ToLower(filename), ".xlsx") || strings.HasSuffix(strings.ToLower(filename), ".xls") {
		headers, sampleData, totalRows, err = s.parseExcel(file)
	} else {
		return nil, fmt.Errorf("unsupported file format")
	}

	if err != nil {
		return nil, err
	}

	return &models.ImportPreview{
		Headers:      headers,
		SampleData:   sampleData,
		TotalRows:    totalRows,
		ModuleFields: module.Fields,
	}, nil
}

func (s *ImportServiceImpl) parseCSV(file io.Reader) ([]string, []map[string]interface{}, int, error) {
	reader := csv.NewReader(file)

	// Read header
	headers, err := reader.Read()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	var sampleData []map[string]interface{}
	totalRows := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to read CSV row: %w", err)
		}

		totalRows++

		// Only keep first 5 rows for preview
		if totalRows <= 5 {
			row := make(map[string]interface{})
			for i, value := range record {
				if i < len(headers) {
					row[headers[i]] = value
				}
			}
			sampleData = append(sampleData, row)
		}
	}

	return headers, sampleData, totalRows, nil
}

func (s *ImportServiceImpl) parseExcel(file io.Reader) ([]string, []map[string]interface{}, int, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	// Get the first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, 0, fmt.Errorf("no sheets found in Excel file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to read Excel rows: %w", err)
	}

	if len(rows) == 0 {
		return nil, nil, 0, fmt.Errorf("Excel file is empty")
	}

	// First row is headers
	headers := rows[0]
	var sampleData []map[string]interface{}
	totalRows := len(rows) - 1 // Excluding header

	// Get sample data (first 5 rows)
	for i := 1; i < len(rows) && i <= 5; i++ {
		row := make(map[string]interface{})
		for j, cell := range rows[i] {
			if j < len(headers) {
				row[headers[j]] = cell
			}
		}
		sampleData = append(sampleData, row)
	}

	return headers, sampleData, totalRows, nil
}

func (s *ImportServiceImpl) ProcessImport(ctx context.Context, jobID string, userID primitive.ObjectID) error {
	job, err := s.ImportRepo.Get(ctx, jobID)
	if err != nil {
		return err
	}

	if job.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Update status to processing
	s.ImportRepo.UpdateStatus(ctx, jobID, models.ImportStatusProcessing)

	// Open the saved file
	file, err := os.Open(job.FilePath)
	if err != nil {
		job.Status = models.ImportStatusFailed
		job.Errors = []models.ImportError{{Row: 0, Field: "file", Message: fmt.Sprintf("Failed to open file: %v", err)}}
		s.ImportRepo.Update(ctx, jobID, job)
		return err
	}
	defer file.Close()

	// Parse file based on extension
	var allData []map[string]interface{}
	var headers []string

	if strings.HasSuffix(strings.ToLower(job.FileName), ".csv") {
		headers, allData, _, err = s.parseCSVFull(file)
	} else if strings.HasSuffix(strings.ToLower(job.FileName), ".xlsx") {
		headers, allData, _, err = s.parseExcelFull(file)
	} else {
		job.Status = models.ImportStatusFailed
		job.Errors = []models.ImportError{{Row: 0, Field: "file", Message: "Unsupported file format"}}
		s.ImportRepo.Update(ctx, jobID, job)
		return fmt.Errorf("unsupported file format")
	}

	if err != nil {
		job.Status = models.ImportStatusFailed
		job.Errors = []models.ImportError{{Row: 0, Field: "file", Message: err.Error()}}
		s.ImportRepo.Update(ctx, jobID, job)
		return err
	}

	// Process each row
	var successCount, errorCount int
	var errors []models.ImportError

	for i, row := range allData {
		// Map columns to module fields using column mapping
		record := make(map[string]interface{})
		for _, header := range headers {
			if fieldName, ok := job.ColumnMapping[header]; ok && fieldName != "" {
				if value, exists := row[header]; exists && value != nil && value != "" {
					record[fieldName] = value
				}
			}
		}

		// Skip empty records
		if len(record) == 0 {
			continue
		}

		// Create record with system context (bypass field permissions for imports)
		_, err := s.RecordService.CreateRecord(ctx, job.ModuleName, record, primitive.NilObjectID)
		if err != nil {
			errorCount++
			errors = append(errors, models.ImportError{
				Row:     i + 2, // +2 because: +1 for header, +1 for 1-indexed
				Field:   "",
				Message: err.Error(),
			})
		} else {
			successCount++
		}

		// Update progress periodically
		if (i+1)%100 == 0 {
			job.ProcessedRecords = i + 1
			job.SuccessCount = successCount
			job.ErrorCount = errorCount
			s.ImportRepo.Update(ctx, jobID, job)
		}
	}

	// Final update
	job.ProcessedRecords = len(allData)
	job.SuccessCount = successCount
	job.ErrorCount = errorCount
	job.Errors = errors
	job.Status = models.ImportStatusCompleted
	now := time.Now()
	job.CompletedAt = &now

	return s.ImportRepo.Update(ctx, jobID, job)
}

// parseCSVFull reads all rows from CSV
func (s *ImportServiceImpl) parseCSVFull(file io.Reader) ([]string, []map[string]interface{}, int, error) {
	reader := csv.NewReader(file)

	headers, err := reader.Read()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	var allData []map[string]interface{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to read CSV row: %w", err)
		}

		row := make(map[string]interface{})
		for i, value := range record {
			if i < len(headers) {
				row[headers[i]] = value
			}
		}
		allData = append(allData, row)
	}

	return headers, allData, len(allData), nil
}

// parseExcelFull reads all rows from Excel
func (s *ImportServiceImpl) parseExcelFull(file io.Reader) ([]string, []map[string]interface{}, int, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, 0, fmt.Errorf("no sheets found in Excel file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to read Excel rows: %w", err)
	}

	if len(rows) == 0 {
		return nil, nil, 0, fmt.Errorf("Excel file is empty")
	}

	headers := rows[0]
	var allData []map[string]interface{}

	for i := 1; i < len(rows); i++ {
		row := make(map[string]interface{})
		for j, cell := range rows[i] {
			if j < len(headers) {
				row[headers[j]] = cell
			}
		}
		allData = append(allData, row)
	}

	return headers, allData, len(allData), nil
}

// ProcessImportWithData processes import with provided data (for immediate processing)
func (s *ImportServiceImpl) ProcessImportWithData(ctx context.Context, data []map[string]interface{}, columnMapping map[string]string, moduleName string, userID primitive.ObjectID, jobID string) error {
	job, err := s.ImportRepo.Get(ctx, jobID)
	if err != nil {
		return err
	}

	s.ImportRepo.UpdateStatus(ctx, jobID, models.ImportStatusProcessing)

	var successCount, errorCount int
	var errors []models.ImportError

	for i, row := range data {
		record := make(map[string]interface{})
		for csvCol, fieldName := range columnMapping {
			if value, ok := row[csvCol]; ok && value != nil && value != "" {
				record[fieldName] = value
			}
		}

		// Create record with system context (bypass field permissions)
		_, err := s.RecordService.CreateRecord(ctx, moduleName, record, primitive.NilObjectID)
		if err != nil {
			errorCount++
			errors = append(errors, models.ImportError{
				Row:     i + 1,
				Field:   "",
				Message: err.Error(),
			})
		} else {
			successCount++
		}

		job.ProcessedRecords = i + 1
		job.SuccessCount = successCount
		job.ErrorCount = errorCount
		job.Errors = errors
	}

	job.Status = models.ImportStatusCompleted
	now := time.Now()
	job.CompletedAt = &now

	return s.ImportRepo.Update(ctx, jobID, job)
}
