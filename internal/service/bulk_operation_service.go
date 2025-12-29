package service

import (
	"context"
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BulkOperationService interface {
	PreviewBulkUpdate(ctx context.Context, moduleName string, filters map[string]interface{}, userID primitive.ObjectID) ([]map[string]any, int, error)
	CreateBulkOperation(ctx context.Context, op *models.BulkOperation) error
	ExecuteBulkUpdate(ctx context.Context, opID string, userID primitive.ObjectID) error
	GetOperation(ctx context.Context, id string) (*models.BulkOperation, error)
	GetUserOperations(ctx context.Context, userID primitive.ObjectID) ([]models.BulkOperation, error)
}

type BulkOperationServiceImpl struct {
	BulkRepo      repository.BulkOperationRepository
	RecordService RecordService
	AuditService  AuditService
}

func NewBulkOperationService(
	bulkRepo repository.BulkOperationRepository,
	recordService RecordService,
	auditService AuditService,
) BulkOperationService {
	return &BulkOperationServiceImpl{
		BulkRepo:      bulkRepo,
		RecordService: recordService,
		AuditService:  auditService,
	}
}

func (s *BulkOperationServiceImpl) PreviewBulkUpdate(ctx context.Context, moduleName string, filters map[string]interface{}, userID primitive.ObjectID) ([]map[string]any, int, error) {
	// Fetch records matching filters
	records, total, err := s.RecordService.ListRecords(ctx, moduleName, filters, 1, 100, "created_at", "desc", userID)
	if err != nil {
		return nil, 0, err
	}

	return records, int(total), nil
}

func (s *BulkOperationServiceImpl) CreateBulkOperation(ctx context.Context, op *models.BulkOperation) error {
	return s.BulkRepo.Create(ctx, op)
}

func (s *BulkOperationServiceImpl) GetOperation(ctx context.Context, id string) (*models.BulkOperation, error) {
	return s.BulkRepo.Get(ctx, id)
}

func (s *BulkOperationServiceImpl) GetUserOperations(ctx context.Context, userID primitive.ObjectID) ([]models.BulkOperation, error) {
	return s.BulkRepo.FindByUserID(ctx, userID.Hex(), 50)
}

func (s *BulkOperationServiceImpl) ExecuteBulkUpdate(ctx context.Context, opID string, userID primitive.ObjectID) error {
	op, err := s.BulkRepo.Get(ctx, opID)
	if err != nil {
		return err
	}

	if op.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Update status to processing
	s.BulkRepo.UpdateStatus(ctx, opID, models.BulkStatusProcessing)

	// Get all records matching filters
	records, total, err := s.RecordService.ListRecords(ctx, op.ModuleName, op.Filters, 1, 10000, "created_at", "desc", primitive.NilObjectID)
	if err != nil {
		op.Status = models.BulkStatusFailed
		op.Errors = []models.BulkError{{RecordID: "", Message: err.Error()}}
		s.BulkRepo.Update(ctx, op)
		return err
	}

	op.TotalRecords = int(total)
	var successCount, errorCount int
	var errors []models.BulkError

	// Update each record
	for _, record := range records {
		recordID := ""
		if id, ok := record["_id"].(primitive.ObjectID); ok {
			recordID = id.Hex()
		} else if id, ok := record["id"].(string); ok {
			recordID = id
		}

		err := s.RecordService.UpdateRecord(ctx, op.ModuleName, recordID, op.Updates, primitive.NilObjectID)
		if err != nil {
			errorCount++
			errors = append(errors, models.BulkError{
				RecordID: recordID,
				Message:  err.Error(),
			})
		} else {
			successCount++

			// Audit log
			changes := make(map[string]models.Change)
			for field, newVal := range op.Updates {
				changes[field] = models.Change{
					Old: record[field],
					New: newVal,
				}
			}
			s.AuditService.LogChange(ctx, models.AuditActionUpdate, op.ModuleName, recordID, changes)
		}

		op.ProcessedCount++
		op.SuccessCount = successCount
		op.ErrorCount = errorCount

		// Update progress every 10 records
		if op.ProcessedCount%10 == 0 {
			s.BulkRepo.Update(ctx, op)
		}
	}

	// Final update
	op.SuccessCount = successCount
	op.ErrorCount = errorCount
	op.Errors = errors
	op.Status = models.BulkStatusCompleted
	now := time.Now()
	op.CompletedAt = &now

	return s.BulkRepo.Update(ctx, op)
}
