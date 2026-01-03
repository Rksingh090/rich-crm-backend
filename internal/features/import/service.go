package import_feature

import (
	"context"
	"encoding/csv"
	"fmt"
	"go-crm/internal/features/module"
	"go-crm/internal/features/record"
	"io"
	"os"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ImportService interface {
	CreateJob(ctx context.Context, job *ImportJob) error
	GetJob(ctx context.Context, id string) (*ImportJob, error)
	GetUserJobs(ctx context.Context, userID primitive.ObjectID) ([]ImportJob, error)
	PreviewFile(ctx context.Context, file io.Reader, filename string, moduleName string) (*ImportPreview, error)
	ProcessImport(ctx context.Context, jobID string, userID primitive.ObjectID) error
	ProcessImportWithData(ctx context.Context, data []map[string]interface{}, columnMapping map[string]string, moduleName string, userID primitive.ObjectID, jobID string) error
}

type ImportServiceImpl struct {
	ImportRepo    ImportRepository
	RecordService record.RecordService
	ModuleService module.ModuleService
}

func NewImportService(
	importRepo ImportRepository,
	recordService record.RecordService,
	moduleService module.ModuleService,
) ImportService {
	return &ImportServiceImpl{
		ImportRepo:    importRepo,
		RecordService: recordService,
		ModuleService: moduleService,
	}
}

func (s *ImportServiceImpl) CreateJob(ctx context.Context, job *ImportJob) error {
	return s.ImportRepo.Create(ctx, job)
}

func (s *ImportServiceImpl) GetJob(ctx context.Context, id string) (*ImportJob, error) {
	return s.ImportRepo.Get(ctx, id)
}

func (s *ImportServiceImpl) GetUserJobs(ctx context.Context, userID primitive.ObjectID) ([]ImportJob, error) {
	return s.ImportRepo.FindByUserID(ctx, userID.Hex(), 50)
}

func (s *ImportServiceImpl) PreviewFile(ctx context.Context, file io.Reader, filename string, moduleName string) (*ImportPreview, error) {
	mod, err := s.ModuleService.GetModuleByName(ctx, moduleName, primitive.NilObjectID)
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

	return &ImportPreview{
		Headers:      headers,
		SampleData:   sampleData,
		TotalRows:    totalRows,
		ModuleFields: mod.Fields,
	}, nil
}

func (s *ImportServiceImpl) parseCSV(file io.Reader) ([]string, []map[string]interface{}, int, error) {
	reader := csv.NewReader(file)

	headers, err := reader.Read()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	var sampleData []map[string]interface{}
	totalRows := 0

	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to read CSV row: %w", err)
		}

		totalRows++

		if totalRows <= 5 {
			row := make(map[string]interface{})
			for i, value := range rec {
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
	var sampleData []map[string]interface{}
	totalRows := len(rows) - 1

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

	s.ImportRepo.UpdateStatus(ctx, jobID, ImportStatusProcessing)

	file, err := os.Open(job.FilePath)
	if err != nil {
		job.Status = ImportStatusFailed
		job.Errors = []ImportError{{Row: 0, Field: "file", Message: fmt.Sprintf("Failed to open file: %v", err)}}
		s.ImportRepo.Update(ctx, jobID, job)
		return err
	}
	defer file.Close()

	var allData []map[string]interface{}
	var headers []string

	if strings.HasSuffix(strings.ToLower(job.FileName), ".csv") {
		headers, allData, _, err = s.parseCSVFull(file)
	} else if strings.HasSuffix(strings.ToLower(job.FileName), ".xlsx") {
		headers, allData, _, err = s.parseExcelFull(file)
	} else {
		job.Status = ImportStatusFailed
		job.Errors = []ImportError{{Row: 0, Field: "file", Message: "Unsupported file format"}}
		s.ImportRepo.Update(ctx, jobID, job)
		return fmt.Errorf("unsupported file format")
	}

	if err != nil {
		job.Status = ImportStatusFailed
		job.Errors = []ImportError{{Row: 0, Field: "file", Message: err.Error()}}
		s.ImportRepo.Update(ctx, jobID, job)
		return err
	}

	var successCount, errorCount int
	var errs []ImportError

	for i, row := range allData {
		rec := make(map[string]interface{})
		for _, header := range headers {
			if fieldName, ok := job.ColumnMapping[header]; ok && fieldName != "" {
				if value, exists := row[header]; exists && value != nil && value != "" {
					rec[fieldName] = value
				}
			}
		}

		if len(rec) == 0 {
			continue
		}

		_, err := s.RecordService.CreateRecord(ctx, job.ModuleName, rec, primitive.NilObjectID)
		if err != nil {
			errorCount++
			errs = append(errs, ImportError{
				Row:     i + 2,
				Field:   "",
				Message: err.Error(),
			})
		} else {
			successCount++
		}

		if (i+1)%100 == 0 {
			job.ProcessedRecords = i + 1
			job.SuccessCount = successCount
			job.ErrorCount = errorCount
			s.ImportRepo.Update(ctx, jobID, job)
		}
	}

	job.ProcessedRecords = len(allData)
	job.SuccessCount = successCount
	job.ErrorCount = errorCount
	job.Errors = errs
	job.Status = ImportStatusCompleted
	now := time.Now()
	job.CompletedAt = &now

	return s.ImportRepo.Update(ctx, jobID, job)
}

func (s *ImportServiceImpl) parseCSVFull(file io.Reader) ([]string, []map[string]interface{}, int, error) {
	reader := csv.NewReader(file)

	headers, err := reader.Read()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	var allData []map[string]interface{}
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to read CSV row: %w", err)
		}

		row := make(map[string]interface{})
		for i, value := range rec {
			if i < len(headers) {
				row[headers[i]] = value
			}
		}
		allData = append(allData, row)
	}

	return headers, allData, len(allData), nil
}

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

func (s *ImportServiceImpl) ProcessImportWithData(ctx context.Context, data []map[string]interface{}, columnMapping map[string]string, moduleName string, userID primitive.ObjectID, jobID string) error {
	job, err := s.ImportRepo.Get(ctx, jobID)
	if err != nil {
		return err
	}

	s.ImportRepo.UpdateStatus(ctx, jobID, ImportStatusProcessing)

	var successCount, errorCount int
	var errs []ImportError

	for i, row := range data {
		rec := make(map[string]interface{})
		for csvCol, fieldName := range columnMapping {
			if value, ok := row[csvCol]; ok && value != nil && value != "" {
				rec[fieldName] = value
			}
		}

		_, err := s.RecordService.CreateRecord(ctx, moduleName, rec, primitive.NilObjectID)
		if err != nil {
			errorCount++
			errs = append(errs, ImportError{
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
		job.Errors = errs
	}

	job.Status = ImportStatusCompleted
	now := time.Now()
	job.CompletedAt = &now

	return s.ImportRepo.Update(ctx, jobID, job)
}
