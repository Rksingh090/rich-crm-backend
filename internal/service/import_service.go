package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ImportService interface {
	CreateJob(ctx context.Context, job *models.ImportJob) error
	GetJob(ctx context.Context, id string) (*models.ImportJob, error)
	GetUserJobs(ctx context.Context, userID primitive.ObjectID) ([]models.ImportJob, error)
	PreviewFile(ctx context.Context, file io.Reader, filename string, moduleName string) (*models.ImportPreview, error)
	ProcessImport(ctx context.Context, jobID string, userID primitive.ObjectID) error
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

	// Parse file based on extension
	// This would typically read from job.FilePath
	// For now, we'll return an error as file handling needs to be implemented

	return fmt.Errorf("import processing not fully implemented - file system integration needed")
}
