package record

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/file"
	"go-crm/internal/features/module"
	"go-crm/internal/features/role"
	"go-crm/internal/features/user"
	"go-crm/internal/features/webhook"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RecordService interface {
	CreateRecord(ctx context.Context, moduleName string, data map[string]interface{}, userID primitive.ObjectID) (interface{}, error)
	GetRecord(ctx context.Context, moduleName, id string, userID primitive.ObjectID) (map[string]any, error)
	ListRecords(ctx context.Context, moduleName string, filters []common_models.Filter, page, limit int64, sortBy string, sortOrder string, userID primitive.ObjectID) ([]map[string]any, int64, error)
	UpdateRecord(ctx context.Context, moduleName, id string, data map[string]interface{}, userID primitive.ObjectID) error
	DeleteRecord(ctx context.Context, moduleName, id string, userID primitive.ObjectID) error
}

// Internal interfaces to break circular dependencies
type AutomationTrigger interface {
	ExecuteFromTrigger(ctx context.Context, moduleName string, record map[string]interface{}, triggerType string) error
}

type ApprovalTrigger interface {
	InitializeApproval(ctx context.Context, moduleName string, record map[string]interface{}) (*common_models.ApprovalRecordState, error)
}

type RecordServiceImpl struct {
	ModuleRepo        module.ModuleRepository
	RecordRepo        RecordRepository
	FileRepo          file.FileRepository
	UserRepo          user.UserRepository
	RoleRepo          role.RoleRepository
	RoleService       role.RoleService
	AuditService      audit.AuditService
	ApprovalService   ApprovalTrigger
	AutomationService AutomationTrigger
	WebhookService    webhook.WebhookService
}

func NewRecordService(
	moduleRepo module.ModuleRepository,
	recordRepo RecordRepository,
	fileRepo file.FileRepository,
	userRepo user.UserRepository,
	roleRepo role.RoleRepository,
	roleService role.RoleService,
	auditService audit.AuditService,
	approvalService ApprovalTrigger,
	automationService AutomationTrigger,
	webhookService webhook.WebhookService,
) RecordService {
	return &RecordServiceImpl{
		ModuleRepo:        moduleRepo,
		RecordRepo:        recordRepo,
		FileRepo:          fileRepo,
		UserRepo:          userRepo,
		RoleRepo:          roleRepo,
		RoleService:       roleService,
		AuditService:      auditService,
		ApprovalService:   approvalService,
		AutomationService: automationService,
		WebhookService:    webhookService,
	}
}

func (s *RecordServiceImpl) CreateRecord(ctx context.Context, moduleName string, data map[string]interface{}, userID primitive.ObjectID) (interface{}, error) {
	// 1. Fetch Schema
	m, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return nil, errors.New("module not found")
	}

	// 2. Validate Data
	validatedData := make(map[string]interface{})
	validatedData["_id"] = primitive.NewObjectID()
	validatedData["created_at"] = time.Now()
	validatedData["updated_at"] = time.Now()
	validatedData["created_by"] = userID // System field - immutable
	validatedData["owner"] = userID      // Mutable field - can be changed

	// Fetch Field Permissions
	perms, _ := s.RoleService.GetFieldPermissions(ctx, userID, moduleName)

	for _, field := range m.Fields {
		val, exists := data[field.Name]

		// Check Required
		if field.Required && (!exists || val == nil || val == "") {
			return nil, fmt.Errorf("field '%s' is required", field.Label)
		}

		if !exists {
			continue // Skip optional missing fields
		}

		// Check Field Permissions
		if perms != nil {
			if p, ok := perms[field.Name]; ok {
				if p == role.FieldPermReadOnly || p == role.FieldPermNone {
					return nil, fmt.Errorf("field '%s' is read-only or hidden", field.Label)
				}
			}
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
	}

	// 4. Insert
	res, err := s.RecordRepo.Create(ctx, moduleName, m.Product, validatedData)
	if err != nil {
		return nil, err
	}

	// 4. Audit Log
	if oid, ok := validatedData["_id"].(primitive.ObjectID); ok {
		changes := make(map[string]common_models.Change)
		for k, v := range validatedData {
			changes[k] = common_models.Change{New: v}
		}
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionCreate, moduleName, oid.Hex(), changes)

		// 5. Automation Trigger
		go func() {
			mergedRecord := make(map[string]interface{})
			for k, v := range validatedData {
				mergedRecord[k] = v
			}

			_ = s.AutomationService.ExecuteFromTrigger(context.Background(), moduleName, validatedData, "create")

			// Webhook
			s.WebhookService.Trigger(context.Background(), "record.updated", common_models.WebhookPayload{
				Event:     "record.created",
				Module:    moduleName,
				RecordID:  validatedData["_id"].(primitive.ObjectID).Hex(),
				Data:      mergedRecord,
				Timestamp: time.Now(),
			})
		}()
	}

	return res, nil
}

func (s *RecordServiceImpl) GetRecord(ctx context.Context, moduleName, id string, userID primitive.ObjectID) (map[string]any, error) {
	record, err := s.RecordRepo.Get(ctx, moduleName, id)
	if err != nil {
		return nil, err
	}

	// Fetch Schema to identify file fields
	m, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return nil, errors.New("module not found")
	}

	// Populate Files
	if err := s.populateFiles(ctx, m.Fields, record); err != nil {
		return nil, err
	}

	// Populate Lookups
	if err := s.populateLookups(ctx, m.Fields, record); err != nil {
		return nil, err
	}

	// Apply Field Permissions
	perms, err := s.RoleService.GetFieldPermissions(ctx, userID, moduleName)
	if err == nil && perms != nil {
		for field, p := range perms {
			if p == role.FieldPermNone {
				delete(record, field)
			}
		}
	}

	return record, nil
}

func (s *RecordServiceImpl) ListRecords(ctx context.Context, moduleName string, filters []common_models.Filter, page, limit int64, sortBy string, sortOrder string, userID primitive.ObjectID) ([]map[string]any, int64, error) {
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
	m, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return nil, 0, errors.New("module not found")
	}

	// 2. Prepare Filters
	typedFilters, err := s.prepareFilters(ctx, m, filters)
	if err != nil {
		return nil, 0, err
	}

	sortOrderInt := -1
	if strings.ToLower(sortOrder) == "asc" {
		sortOrderInt = 1
	}

	// 3. Access Control
	accessFilter, err := s.RoleService.GetAccessFilter(ctx, userID, moduleName, "read")
	if err != nil {
		return nil, 0, err
	}

	records, err := s.RecordRepo.List(ctx, moduleName, typedFilters, accessFilter, limit, offset, sortBy, sortOrderInt)
	if err != nil {
		return nil, 0, err
	}

	for _, record := range records {
		_ = s.populateFiles(ctx, m.Fields, record)
		_ = s.populateLookups(ctx, m.Fields, record)
	}

	totalCount, err := s.RecordRepo.Count(ctx, moduleName, typedFilters, accessFilter)
	if err != nil {
		return nil, 0, err
	}

	perms, err := s.RoleService.GetFieldPermissions(ctx, userID, moduleName)
	if err == nil && perms != nil {
		for _, record := range records {
			for field, p := range perms {
				if p == role.FieldPermNone {
					delete(record, field)
				}
			}
		}
	}

	return records, totalCount, nil
}

func (s *RecordServiceImpl) UpdateRecord(ctx context.Context, moduleName, id string, data map[string]interface{}, userID primitive.ObjectID) error {
	m, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return errors.New("module not found")
	}

	validatedData := make(map[string]interface{})
	validatedData["updated_at"] = time.Now()

	if ownerVal, exists := data["owner"]; exists {
		if ownerStr, ok := ownerVal.(string); ok {
			if ownerID, err := primitive.ObjectIDFromHex(ownerStr); err == nil {
				validatedData["owner"] = ownerID
			}
		}
	}

	perms, _ := s.RoleService.GetFieldPermissions(ctx, userID, moduleName)

	for _, field := range m.Fields {
		val, exists := data[field.Name]
		if !exists {
			continue
		}

		if perms != nil {
			if p, ok := perms[field.Name]; ok {
				if p == role.FieldPermReadOnly || p == role.FieldPermNone {
					return fmt.Errorf("field '%s' is read-only or hidden", field.Label)
				}
			}
		}

		cleanVal, err := s.validateAndConvert(ctx, field, val)
		if err != nil {
			return fmt.Errorf("invalid value for field '%s': %v", field.Label, err)
		}
		validatedData[field.Name] = cleanVal
	}

	oldRecord, err := s.RecordRepo.Get(ctx, moduleName, id)
	if err != nil {
		return err
	}

	if val, ok := oldRecord["_approval"]; ok {
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

	err = s.RecordRepo.Update(ctx, moduleName, id, validatedData)
	if err != nil {
		return err
	}

	changes := make(map[string]common_models.Change)
	for k, newVal := range validatedData {
		oldVal, exists := oldRecord[k]
		if !exists || oldVal != newVal {
			changes[k] = common_models.Change{Old: oldVal, New: newVal}
		}
	}
	if len(changes) > 0 {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionUpdate, moduleName, id, changes)

		go func() {
			mergedRecord := make(map[string]interface{})
			for k, v := range oldRecord {
				mergedRecord[k] = v
			}
			for k, v := range validatedData {
				mergedRecord[k] = v
			}

			_ = s.AutomationService.ExecuteFromTrigger(context.Background(), moduleName, mergedRecord, "update")

			s.WebhookService.Trigger(context.Background(), "record.updated", common_models.WebhookPayload{
				Event:     "record.updated",
				Module:    moduleName,
				RecordID:  id,
				Data:      mergedRecord,
				Timestamp: time.Now(),
			})
		}()
	}
	return nil
}

func (s *RecordServiceImpl) DeleteRecord(ctx context.Context, moduleName, id string, userID primitive.ObjectID) error {
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

	err = s.RecordRepo.Delete(ctx, moduleName, id, userID)
	if err == nil {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionDelete, moduleName, id, nil)
	}
	return err
}

func (s *RecordServiceImpl) populateFiles(ctx context.Context, fields []module.ModuleField, record map[string]any) error {
	for _, field := range fields {
		if field.Type == module.FieldTypeFile || field.Type == module.FieldTypeImage {
			if val, ok := record[field.Name]; ok {
				var idStr string
				if oid, ok := val.(primitive.ObjectID); ok {
					idStr = oid.Hex()
				} else if s, ok := val.(string); ok {
					idStr = s
				}

				if idStr != "" {
					file, err := s.FileRepo.Get(ctx, idStr)
					if err == nil {
						record[field.Name] = map[string]interface{}{
							"id":                file.ID,
							"original_filename": file.OriginalFilename,
							"url":               file.URL,
						}
					}
				}
			}
		}
	}
	return nil
}

func (s *RecordServiceImpl) populateLookups(ctx context.Context, fields []module.ModuleField, record map[string]any) error {
	for _, field := range fields {
		if field.Type == module.FieldTypeLookup && field.Lookup != nil {
			if val, ok := record[field.Name]; ok {
				var idStr string
				if oid, ok := val.(primitive.ObjectID); ok {
					idStr = oid.Hex()
				} else if s, ok := val.(string); ok {
					idStr = s
				}

				if idStr != "" {
					refRecord, err := s.RecordRepo.Get(ctx, field.Lookup.LookupModule, idStr)
					if err == nil {
						displayField := "name"
						if field.Lookup.LookupLabel != "" {
							displayField = field.Lookup.LookupLabel
						}

						displayValue, _ := refRecord[displayField]

						record[field.Name] = map[string]interface{}{
							"id":   idStr,
							"name": displayValue,
						}
					}
				}
			}
		}
	}
	return nil
}

func (s *RecordServiceImpl) validateAndConvert(ctx context.Context, field module.ModuleField, val interface{}) (interface{}, error) {
	if val == nil {
		return nil, nil
	}
	if strVal, ok := val.(string); ok && strVal == "" {
		if field.Type != module.FieldTypeText && field.Type != module.FieldTypeTextArea && field.Type != module.FieldTypeSelect && field.Type != module.FieldTypeMultiSelect {
			return nil, nil
		}
	}

	switch field.Type {
	case module.FieldTypeNumber:
		switch v := val.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case string:
			if v == "" {
				return nil, nil
			}
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, errors.New("expected number")
			}
			return f, nil
		default:
			return nil, errors.New("expected number")
		}
	case module.FieldTypeBoolean:
		if b, ok := val.(bool); ok {
			return b, nil
		}
		if s, ok := val.(string); ok {
			if s == "" {
				return nil, nil
			}
			b, err := strconv.ParseBool(s)
			if err != nil {
				return nil, errors.New("expected boolean")
			}
			return b, nil
		}
		return nil, errors.New("expected boolean")
	case module.FieldTypeDate:
		strVal, ok := val.(string)
		if !ok {
			return nil, errors.New("expected date string")
		}
		if strVal == "" {
			return nil, nil
		}
		t, err := time.Parse(time.RFC3339, strVal)
		if err != nil {
			t, err = time.Parse("2006-01-02", strVal)
			if err != nil {
				return nil, errors.New("invalid date format (use RFC3339 or YYYY-MM-DD)")
			}
		}
		return t, nil
	case module.FieldTypeEmail:
		strVal, ok := val.(string)
		if !ok {
			return nil, errors.New("expected string")
		}
		if strVal == "" {
			return nil, nil
		}
		if match, _ := regexp.MatchString(`^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$`, strVal); !match {
			return nil, errors.New("invalid email format")
		}
		return strVal, nil
	case module.FieldTypeLookup:
		strVal, ok := val.(string)
		if !ok {
			return nil, errors.New("expected valid objectID hex string")
		}
		if strVal == "" {
			return nil, nil
		}
		oid, err := primitive.ObjectIDFromHex(strVal)
		if err != nil {
			return nil, errors.New("invalid objectID for lookup")
		}

		if field.Lookup != nil && field.Lookup.LookupModule != "" {
			_, err := s.RecordRepo.Get(ctx, field.Lookup.LookupModule, strVal)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					return nil, fmt.Errorf("referenced record in module '%s' not found", field.Lookup.LookupModule)
				}
				return nil, fmt.Errorf("failed to verify lookup reference: %v", err)
			}
		}

		return oid, nil
	case module.FieldTypeFile:
		strVal, ok := val.(string)
		if !ok {
			return nil, errors.New("expected string for file")
		}
		if strVal == "" {
			return nil, nil
		}

		if _, err := primitive.ObjectIDFromHex(strVal); err == nil {
			_, err = s.FileRepo.Get(ctx, strVal)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					return nil, errors.New("referenced file not found")
				}
				return nil, fmt.Errorf("failed to verify file reference: %v", err)
			}
			return strVal, nil
		}

		return strVal, nil

	case module.FieldTypeImage:
		strVal, ok := val.(string)
		if !ok {
			return nil, errors.New("expected string for image")
		}
		return strVal, nil
	default:
		return val, nil
	}
}

func (s *RecordServiceImpl) prepareFilters(ctx context.Context, m *common_models.Entity, filters []common_models.Filter) (bson.M, error) {
	typedFilters := bson.M{}

	for _, f := range filters {
		fieldName := f.Field
		operator := f.Operator
		val := f.Value

		// Handle Special ID fields
		if fieldName == "id" || fieldName == "_id" {
			if operator == "in" {
				var ids []primitive.ObjectID
				switch v := val.(type) {
				case string:
					for _, p := range strings.Split(v, ",") {
						if oid, err := primitive.ObjectIDFromHex(strings.TrimSpace(p)); err == nil {
							ids = append(ids, oid)
						}
					}
				case []string:
					for _, s := range v {
						if oid, err := primitive.ObjectIDFromHex(s); err == nil {
							ids = append(ids, oid)
						}
					}
				case []interface{}:
					for _, item := range v {
						if s, ok := item.(string); ok {
							if oid, err := primitive.ObjectIDFromHex(s); err == nil {
								ids = append(ids, oid)
							}
						}
					}
				}
				if len(ids) > 0 {
					typedFilters["_id"] = bson.M{"$in": ids}
				}
			} else if operator == "" || operator == "eq" {
				if strVal, ok := val.(string); ok {
					if oid, err := primitive.ObjectIDFromHex(strVal); err == nil {
						typedFilters["_id"] = oid
					}
				} else if oid, ok := val.(primitive.ObjectID); ok {
					typedFilters["_id"] = oid
				}
			}
			continue
		}

		// Handle System fields (created_at, updated_at, etc)
		var field *common_models.ModuleField
		for _, fDef := range m.Fields {
			if fDef.Name == fieldName {
				field = &fDef
				break
			}
		}

		if field == nil {
			// If not in schema, it might be a system field or unknown
			typedFilters[fieldName] = val
			continue
		}

		if operator == "between" {
			if strVal, ok := val.(string); ok {
				parts := strings.Split(strVal, ",")
				if len(parts) == 2 {
					startStr := strings.TrimSpace(parts[0])
					endStr := strings.TrimSpace(parts[1])

					startTime, err1 := time.Parse("2006-01-02", startStr)
					endTime, err2 := time.Parse("2006-01-02", endStr)

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
						startFloat, errF1 := strconv.ParseFloat(startStr, 64)
						endFloat, errF2 := strconv.ParseFloat(endStr, 64)
						if errF1 == nil && errF2 == nil {
							typedFilters[fieldName] = bson.M{
								"$gte": startFloat,
								"$lte": endFloat,
							}
						} else {
							return nil, fmt.Errorf("invalid range values for field '%s'", field.Label)
						}
					}
				}
			}
		} else {
			typedVal, err := s.validateAndConvert(ctx, *field, val)
			if err != nil {
				return nil, fmt.Errorf("invalid filter value for '%s': %v", field.Label, err)
			}

			switch operator {
			case "", "eq":
				typedFilters[fieldName] = typedVal
			case "ne":
				typedFilters[fieldName] = bson.M{"$ne": typedVal}
			case "contains":
				if strVal, ok := typedVal.(string); ok {
					typedFilters[fieldName] = bson.M{"$regex": primitive.Regex{Pattern: strVal, Options: "i"}}
				} else {
					typedFilters[fieldName] = typedVal
				}
			case "gt":
				typedFilters[fieldName] = bson.M{"$gt": typedVal}
			case "lt":
				typedFilters[fieldName] = bson.M{"$lt": typedVal}
			case "gte":
				typedFilters[fieldName] = bson.M{"$gte": typedVal}
			case "lte":
				typedFilters[fieldName] = bson.M{"$lte": typedVal}
			case "in":
				typedFilters[fieldName] = bson.M{"$in": typedVal}
			case "nin":
				typedFilters[fieldName] = bson.M{"$nin": typedVal}
			case "starts_with":
				if strVal, ok := typedVal.(string); ok {
					typedFilters[fieldName] = bson.M{"$regex": primitive.Regex{Pattern: "^" + strVal, Options: "i"}}
				}
			case "ends_with":
				if strVal, ok := typedVal.(string); ok {
					typedFilters[fieldName] = bson.M{"$regex": primitive.Regex{Pattern: strVal + "$", Options: "i"}}
				}
			}
		}
	}

	return typedFilters, nil
}
