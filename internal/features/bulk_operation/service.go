package bulk_operation

import (
	"context"
	"fmt"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/record"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BulkOperationService interface {
	PreviewBulkOperation(ctx context.Context, moduleName string, filters map[string]interface{}, userID primitive.ObjectID) ([]map[string]any, int, error)
	CreateBulkOperation(ctx context.Context, op *BulkOperation) error
	ExecuteBulkOperation(ctx context.Context, opID string, userID primitive.ObjectID) error
	GetOperation(ctx context.Context, id string) (*BulkOperation, error)
	GetUserOperations(ctx context.Context, userID primitive.ObjectID) ([]BulkOperation, error)
}

type BulkOperationServiceImpl struct {
	BulkRepo      BulkOperationRepository
	RecordService record.RecordService
	AuditService  audit.AuditService
}

func NewBulkOperationService(
	bulkRepo BulkOperationRepository,
	recordService record.RecordService,
	auditService audit.AuditService,
) BulkOperationService {
	return &BulkOperationServiceImpl{
		BulkRepo:      bulkRepo,
		RecordService: recordService,
		AuditService:  auditService,
	}
}

func (s *BulkOperationServiceImpl) PreviewBulkOperation(ctx context.Context, moduleName string, filters map[string]interface{}, userID primitive.ObjectID) ([]map[string]any, int, error) {
	records, total, err := s.RecordService.ListRecords(ctx, moduleName, filters, 1, 100, "created_at", "desc", userID)
	if err != nil {
		return nil, 0, err
	}

	return records, int(total), nil
}

func (s *BulkOperationServiceImpl) CreateBulkOperation(ctx context.Context, op *BulkOperation) error {
	return s.BulkRepo.Create(ctx, op)
}

func (s *BulkOperationServiceImpl) GetOperation(ctx context.Context, id string) (*BulkOperation, error) {
	return s.BulkRepo.Get(ctx, id)
}

func (s *BulkOperationServiceImpl) GetUserOperations(ctx context.Context, userID primitive.ObjectID) ([]BulkOperation, error) {
	return s.BulkRepo.FindByUserID(ctx, userID.Hex(), 50)
}

func (s *BulkOperationServiceImpl) ExecuteBulkOperation(ctx context.Context, opID string, userID primitive.ObjectID) error {
	op, err := s.BulkRepo.Get(ctx, opID)
	if err != nil {
		return err
	}

	if op.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	s.BulkRepo.UpdateStatus(ctx, opID, BulkStatusProcessing)

	records, total, err := s.RecordService.ListRecords(ctx, op.ModuleName, op.Filters, 1, 10000, "created_at", "desc", primitive.NilObjectID)
	if err != nil {
		op.Status = BulkStatusFailed
		op.Errors = []BulkError{{RecordID: "", Message: err.Error()}}
		s.BulkRepo.Update(ctx, op)
		return err
	}

	op.TotalRecords = int(total)
	var successCount, errorCount int
	var errs []BulkError

	for _, rec := range records {
		recordID := ""
		if id, ok := rec["_id"].(primitive.ObjectID); ok {
			recordID = id.Hex()
		} else if id, ok := rec["id"].(string); ok {
			recordID = id
		}

		var err error
		if op.Type == BulkTypeDelete {
			err = s.RecordService.DeleteRecord(ctx, op.ModuleName, recordID)
		} else {
			err = s.RecordService.UpdateRecord(ctx, op.ModuleName, recordID, op.Updates, primitive.NilObjectID)
		}

		if err != nil {
			errorCount++
			errs = append(errs, BulkError{
				RecordID: recordID,
				Message:  err.Error(),
			})
		} else {
			successCount++
		}

		op.ProcessedCount++
		op.SuccessCount = successCount
		op.ErrorCount = errorCount

		if op.ProcessedCount%10 == 0 {
			s.BulkRepo.Update(ctx, op)
		}
	}

	op.SuccessCount = successCount
	op.ErrorCount = errorCount
	op.Errors = errs
	op.Status = BulkStatusCompleted
	now := time.Now()
	op.CompletedAt = &now

	return s.BulkRepo.Update(ctx, op)
}
