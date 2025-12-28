package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RecordService interface {
	CreateRecord(ctx context.Context, moduleName string, data map[string]interface{}) (interface{}, error)
	GetRecord(ctx context.Context, moduleName, id string) (map[string]any, error)
	ListRecords(ctx context.Context, moduleName string, filters map[string]any, page, limit int64, sortBy string, sortOrder string) ([]map[string]any, int64, error)
	UpdateRecord(ctx context.Context, moduleName, id string, data map[string]interface{}) error
	DeleteRecord(ctx context.Context, moduleName, id string) error
}

type RecordServiceImpl struct {
	ModuleRepo        repository.ModuleRepository
	RecordRepo        repository.RecordRepository
	FileRepo          repository.FileRepository
	AuditService      AuditService
	ApprovalService   ApprovalService
	AutomationService AutomationService
}

func NewRecordService(moduleRepo repository.ModuleRepository, recordRepo repository.RecordRepository, fileRepo repository.FileRepository, auditService AuditService, approvalService ApprovalService, automationService AutomationService) RecordService {
	return &RecordServiceImpl{
		ModuleRepo:        moduleRepo,
		RecordRepo:        recordRepo,
		FileRepo:          fileRepo,
		AuditService:      auditService,
		ApprovalService:   approvalService,
		AutomationService: automationService,
	}
}

func (s *RecordServiceImpl) CreateRecord(ctx context.Context, moduleName string, data map[string]interface{}) (interface{}, error) {
	// 1. Fetch Schema
	module, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return nil, errors.New("module not found")
	}

	// 2. Validate Data
	validatedData := make(map[string]interface{})
	validatedData["_id"] = primitive.NewObjectID()
	validatedData["created_at"] = time.Now()
	validatedData["updated_at"] = time.Now()

	for _, field := range module.Fields {
		val, exists := data[field.Name]

		// Check Required
		if field.Required && (!exists || val == nil || val == "") {
			return nil, fmt.Errorf("field '%s' is required", field.Label)
		}

		if !exists {
			continue // Skip optional missing fields
		}

		// Validate Type
		cleanVal, err := s.validateAndConvert(ctx, field, val)
		if err != nil {
			return nil, fmt.Errorf("invalid value for field '%s': %v", field.Label, err)
		}
		validatedData[field.Name] = cleanVal
	}

	// 3. Initialize Approval Workflow
	approvalState, err := s.ApprovalService.InitializeApproval(ctx, moduleName, validatedData)
	if err != nil {
		return nil, fmt.Errorf("failed to check approval workflow: %v", err)
	}
	if approvalState != nil {
		validatedData["_approval"] = approvalState
		// If using statuses, maybe override "status" field too?
		// validatedData["status"] = "pending_approval"
	}

	// 4. Insert
	res, err := s.RecordRepo.Create(ctx, moduleName, validatedData)
	if err != nil {
		return nil, err
	}

	// 4. Audit Log
	if oid, ok := validatedData["_id"].(primitive.ObjectID); ok {
		changes := make(map[string]models.Change)
		for k, v := range validatedData {
			changes[k] = models.Change{New: v}
		}
		_ = s.AuditService.LogChange(ctx, models.AuditActionCreate, moduleName, oid.Hex(), changes)

		// 5. Automation Trigger (Async optional, but sync for now)
		// Pass the full record (including generated ID)
		// Note: validatedData has the raw values. Automation might expect formatted/lookup objects?
		// AutomationService expects map[string]interface
		go func() {
			// Use background context or detached context if needed, but for now simple go routine with current context
			// might be risky if context cancels. Better to create new context in real app.
			_ = s.AutomationService.ExecuteFromTrigger(context.Background(), moduleName, validatedData, "create")
		}()
	}

	return res, nil
}

func (s *RecordServiceImpl) GetRecord(ctx context.Context, moduleName, id string) (map[string]any, error) {
	record, err := s.RecordRepo.Get(ctx, moduleName, id)
	if err != nil {
		return nil, err
	}

	// Fetch Schema to identify file fields
	module, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return nil, errors.New("module not found")
	}

	// Populate Files
	if err := s.populateFiles(ctx, module.Fields, record); err != nil {
		return nil, err
	}

	// Populate Lookups
	if err := s.populateLookups(ctx, module.Fields, record); err != nil {
		return nil, err
	}

	return record, nil
}

func (s *RecordServiceImpl) ListRecords(ctx context.Context, moduleName string, filters map[string]any, page, limit int64, sortBy string, sortOrder string) ([]map[string]any, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// 1. Fetch Schema to handle type conversion for filters
	module, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return nil, 0, errors.New("module not found")
	}

	// 2. Convert Filters
	typedFilters := make(map[string]interface{})
	for k, v := range filters {
		fieldName := k
		operator := ""

		// Check for operators (suffix convention: field__op)
		if strings.Contains(k, "__") {
			parts := strings.Split(k, "__")
			if len(parts) == 2 {
				fieldName = parts[0]
				operator = parts[1]
			}
		}

		var field *models.ModuleField
		for _, f := range module.Fields {
			if f.Name == fieldName {
				field = &f
				break
			}
		}

		if field != nil {
			// Special handling for 'between' operator before general validateAndConvert
			if operator == "between" {
				// Expecting val to be string "start,end"
				if strVal, ok := v.(string); ok {
					parts := strings.Split(strVal, ",")
					if len(parts) == 2 {
						startStr := strings.TrimSpace(parts[0])
						endStr := strings.TrimSpace(parts[1])

						// Try to parse dates
						startTime, err1 := time.Parse("2006-01-02", startStr)
						endTime, err2 := time.Parse("2006-01-02", endStr)

						// If YYYY-MM-DD fails, try RFC3339
						if err1 != nil {
							startTime, err1 = time.Parse(time.RFC3339, startStr)
						}
						if err2 != nil {
							endTime, err2 = time.Parse(time.RFC3339, endStr)
						}

						if err1 == nil && err2 == nil {
							typedFilters[fieldName] = bson.M{
								"$gte": startTime,
								"$lte": endTime,
							}
						} else {
							// If parsing fails for Date fields, maybe it's Number or something else?
							// Fallback: try float for numbers?
							startFloat, errF1 := strconv.ParseFloat(startStr, 64)
							endFloat, errF2 := strconv.ParseFloat(endStr, 64)
							if errF1 == nil && errF2 == nil {
								typedFilters[fieldName] = bson.M{
									"$gte": startFloat,
									"$lte": endFloat,
								}
							} else {
								// Invalid range format
								return nil, 0, fmt.Errorf("invalid range values for field '%s'", k)
							}
						}
					}
				}
			} else {
				typedVal, err := s.validateAndConvert(ctx, *field, v)

				if err == nil {
					switch operator {
					case "":
						typedFilters[fieldName] = typedVal
					case "ne":
						typedFilters[fieldName] = bson.M{"$ne": typedVal}
					case "contains":
						// Contains only makes sense for strings usually
						if strVal, ok := v.(string); ok {
							typedFilters[fieldName] = bson.M{"$regex": primitive.Regex{Pattern: strVal, Options: "i"}}
						} else {
							// e.g. contains for number? not really supported easily
							typedFilters[fieldName] = typedVal // Fallback to eq
						}
					case "gt":
						typedFilters[fieldName] = bson.M{"$gt": typedVal}
					case "lt":
						typedFilters[fieldName] = bson.M{"$lt": typedVal}
					case "gte":
						typedFilters[fieldName] = bson.M{"$gte": typedVal}
					case "lte":
						typedFilters[fieldName] = bson.M{"$lte": typedVal}
					}
				} else {
					return nil, 0, fmt.Errorf("invalid filter value for '%s': %v", k, err)
				}
			}
		}
	}

	// Map sortOrder string ("asc"/"desc") to int (1/-1)
	sortOrderInt := -1
	if strings.ToLower(sortOrder) == "asc" {
		sortOrderInt = 1
	}

	// 3. List Records
	records, err := s.RecordRepo.List(ctx, moduleName, typedFilters, limit, offset, sortBy, sortOrderInt)
	if err != nil {
		return nil, 0, err
	}

	// Populate Files and Lookups for all records
	for _, record := range records {
		_ = s.populateFiles(ctx, module.Fields, record)
		_ = s.populateLookups(ctx, module.Fields, record)
	}

	// 4. Get Total Count
	totalCount, err := s.RecordRepo.Count(ctx, moduleName, typedFilters)
	if err != nil {
		return nil, 0, err
	}

	return records, totalCount, nil
}

func (s *RecordServiceImpl) UpdateRecord(ctx context.Context, moduleName, id string, data map[string]interface{}) error {
	// 1. Fetch Schema
	module, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return errors.New("module not found")
	}

	// 2. Validate Data (Partial update)
	validatedData := make(map[string]interface{})
	validatedData["updated_at"] = time.Now()

	for _, field := range module.Fields {
		val, exists := data[field.Name]
		if !exists {
			continue // Partial update, skip missing fields
		}

		// Validate Type if present
		cleanVal, err := s.validateAndConvert(ctx, field, val)
		if err != nil {
			return fmt.Errorf("invalid value for field '%s': %v", field.Label, err)
		}
		validatedData[field.Name] = cleanVal
	}

	// 3. Check Approval Lock
	oldRecord, err := s.RecordRepo.Get(ctx, moduleName, id)
	if err != nil {
		return err
	}

	if val, ok := oldRecord["_approval"]; ok {
		// Use helper or manual check. Since extractApprovalState is private in ApprovalService,
		// we might need to expose it or just do a quick check here.
		// Or better, let ApprovalService handle permission check? No, this is Update logic.
		// Let's rely on map structure. "Pending" is strict string.
		// Assuming map[string]interface{}.

		// Convert val to map if possible
		if stateMap, ok := val.(map[string]interface{}); ok {
			if status, ok := stateMap["status"].(string); ok && status == "pending" {
				return errors.New("record is locked for approval and cannot be edited")
			}
		} else if stateMap, ok := val.(primitive.M); ok {
			if status, ok := stateMap["status"].(string); ok && status == "pending" {
				return errors.New("record is locked for approval and cannot be edited")
			}
		}
	}

	// 4. Update

	err = s.RecordRepo.Update(ctx, moduleName, id, validatedData)
	if err != nil {
		return err
	}

	// 4. Audit Log
	changes := make(map[string]models.Change)
	for k, newVal := range validatedData {
		oldVal, exists := oldRecord[k]
		// Determine if changed. Simple equality check.
		// Be careful with types (e.g. int vs float).
		if !exists || oldVal != newVal {
			changes[k] = models.Change{Old: oldVal, New: newVal}
		}
	}
	if len(changes) > 0 {
		_ = s.AuditService.LogChange(ctx, models.AuditActionUpdate, moduleName, id, changes)

		// 5. Automation Trigger
		go func() {
			// Need the full updated record for conditions
			// We only have 'validatedData' (partial update) and 'oldRecord'.
			// Construct merged record for evaluation
			mergedRecord := make(map[string]interface{})
			for k, v := range oldRecord {
				mergedRecord[k] = v
			}
			for k, v := range validatedData {
				mergedRecord[k] = v
			}
			_ = s.AutomationService.ExecuteFromTrigger(context.Background(), moduleName, mergedRecord, "update")
		}()
	}
	return nil
}

func (s *RecordServiceImpl) DeleteRecord(ctx context.Context, moduleName, id string) error {
	// 1. Check Approval Lock
	oldRecord, err := s.RecordRepo.Get(ctx, moduleName, id)
	if err != nil {
		return err
	}

	if val, ok := oldRecord["_approval"]; ok {
		if stateMap, ok := val.(map[string]interface{}); ok {
			if status, ok := stateMap["status"].(string); ok && status == "pending" {
				return errors.New("record is locked for approval and cannot be deleted")
			}
		} else if stateMap, ok := val.(primitive.M); ok {
			if status, ok := stateMap["status"].(string); ok && status == "pending" {
				return errors.New("record is locked for approval and cannot be deleted")
			}
		}
	}

	// 2. Delete
	err = s.RecordRepo.Delete(ctx, moduleName, id)

	if err == nil {
		_ = s.AuditService.LogChange(ctx, models.AuditActionDelete, moduleName, id, nil)
	}
	return err
}

func (s *RecordServiceImpl) populateFiles(ctx context.Context, fields []models.ModuleField, record map[string]any) error {
	for _, field := range fields {
		if field.Type == models.FieldTypeFile {
			if val, ok := record[field.Name]; ok {
				// val should be a string (hex ID)
				if idStr, ok := val.(string); ok && idStr != "" {
					file, err := s.FileRepo.Get(ctx, idStr)
					if err == nil {
						// Replace ID with limited File Object
						record[field.Name] = map[string]interface{}{
							"id":                file.ID,
							"original_filename": file.OriginalFilename,
							"url":               file.URL,
						}
					}
					// If error (not found), we keep the ID string or set null?
					// Let's keep the ID string if file not found to avoid data loss in UI
				}
			}
		}
	}
	return nil
}

func (s *RecordServiceImpl) populateLookups(ctx context.Context, fields []models.ModuleField, record map[string]any) error {
	for _, field := range fields {
		if field.Type == models.FieldTypeLookup && field.Lookup != nil {
			if val, ok := record[field.Name]; ok {
				// val should be an ObjectID or string hex
				var idStr string
				if oid, ok := val.(primitive.ObjectID); ok {
					idStr = oid.Hex()
				} else if s, ok := val.(string); ok {
					idStr = s
				}

				if idStr != "" {
					// Fetch Referenced Record
					refRecord, err := s.RecordRepo.Get(ctx, field.Lookup.Module, idStr)
					if err == nil {
						// Determine Display Field
						displayField := "name" // Default
						if field.Lookup.DisplayField != "" {
							displayField = field.Lookup.DisplayField
						}

						displayValue, _ := refRecord[displayField]

						// Replace value with Object
						record[field.Name] = map[string]interface{}{
							"id":   idStr,
							"name": displayValue, // Map to generic "name" key
						}
					}
				}
			}
		}
	}
	return nil
}

func (s *RecordServiceImpl) validateAndConvert(ctx context.Context, field models.ModuleField, val interface{}) (interface{}, error) {
	switch field.Type {
	case models.FieldTypeNumber:
		// ... existing number logic
		switch v := val.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, errors.New("expected number")
			}
			return f, nil
		default:
			return nil, errors.New("expected number")
		}
	case models.FieldTypeBoolean:
		// ... existing boolean logic
		if b, ok := val.(bool); ok {
			return b, nil
		}
		if s, ok := val.(string); ok {
			b, err := strconv.ParseBool(s)
			if err != nil {
				return nil, errors.New("expected boolean")
			}
			return b, nil
		}
		return nil, errors.New("expected boolean")
	case models.FieldTypeDate:
		// ... existing date logic
		strVal, ok := val.(string)
		if !ok {
			return nil, errors.New("expected date string")
		}
		t, err := time.Parse(time.RFC3339, strVal)
		if err != nil {
			t, err = time.Parse("2006-01-02", strVal)
			if err != nil {
				return nil, errors.New("invalid date format (use RFC3339 or YYYY-MM-DD)")
			}
		}
		return t, nil
	case models.FieldTypeEmail:
		// ... existing email logic
		strVal, ok := val.(string)
		if !ok {
			return nil, errors.New("expected string")
		}
		if match, _ := regexp.MatchString(`^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$`, strVal); !match {
			return nil, errors.New("invalid email format")
		}
		return strVal, nil
	case models.FieldTypeLookup:
		strVal, ok := val.(string)
		if !ok {
			return nil, errors.New("expected valid objectID hex string")
		}
		oid, err := primitive.ObjectIDFromHex(strVal)
		if err != nil {
			return nil, errors.New("invalid objectID for lookup")
		}

		// Integrity Check
		if field.Lookup != nil && field.Lookup.Module != "" {
			// Check if referenced record exists
			// We use Get from RecordRepo. Ideally, we might want a simpler "Exists" method, but Get works.
			_, err := s.RecordRepo.Get(ctx, field.Lookup.Module, strVal)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					return nil, fmt.Errorf("referenced record in module '%s' not found", field.Lookup.Module)
				}
				// If other error, we might log it but assume it's okay or fail?
				// Let's be strict: if we can't verify, we assume invalid to maintain integrity.
				// However, if the collection doesn't exist yet, it's definitely invalid reference.
				return nil, fmt.Errorf("failed to verify lookup reference: %v", err)
			}
		}

		return oid, nil
	case models.FieldTypeFile:
		strVal, ok := val.(string)
		if !ok {
			return nil, errors.New("expected valid objectID hex string for file")
		}
		// Validate ID format
		_, err := primitive.ObjectIDFromHex(strVal)
		if err != nil {
			return nil, errors.New("invalid file ID format")
		}

		// Check existence
		// We can reuse Get, passing the ID directly
		_, err = s.FileRepo.Get(ctx, strVal)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, errors.New("referenced file not found")
			}
			return nil, fmt.Errorf("failed to verify file reference: %v", err)
		}
		return strVal, nil
	default:
		return val, nil
	}
}
