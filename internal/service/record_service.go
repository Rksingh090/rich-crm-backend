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
	CreateRecord(ctx context.Context, moduleName string, data map[string]interface{}, userID primitive.ObjectID) (interface{}, error)
	GetRecord(ctx context.Context, moduleName, id string, userID primitive.ObjectID) (map[string]any, error)
	ListRecords(ctx context.Context, moduleName string, filters map[string]any, page, limit int64, sortBy string, sortOrder string, userID primitive.ObjectID) ([]map[string]any, int64, error)
	UpdateRecord(ctx context.Context, moduleName, id string, data map[string]interface{}, userID primitive.ObjectID) error
	DeleteRecord(ctx context.Context, moduleName, id string) error
}

type RecordServiceImpl struct {
	ModuleRepo        repository.ModuleRepository
	RecordRepo        repository.RecordRepository
	FileRepo          repository.FileRepository
	UserRepo          repository.UserRepository
	RoleRepo          repository.RoleRepository
	RoleService       RoleService
	AuditService      AuditService
	ApprovalService   ApprovalService
	AutomationService AutomationService
	WebhookService    WebhookService
}

func NewRecordService(moduleRepo repository.ModuleRepository, recordRepo repository.RecordRepository, fileRepo repository.FileRepository, userRepo repository.UserRepository, roleRepo repository.RoleRepository, roleService RoleService, auditService AuditService, approvalService ApprovalService, automationService AutomationService, webhookService WebhookService) RecordService {
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
	module, err := s.ModuleRepo.FindByName(ctx, moduleName)
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

	for _, field := range module.Fields {
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
				if p == models.FieldPermReadOnly || p == models.FieldPermNone {
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
			// Need the full updated record for conditions
			mergedRecord := make(map[string]interface{})
			for k, v := range validatedData {
				mergedRecord[k] = v
			}

			// Use background context or detached context if needed, but for now simple go routine with current context
			// might be risky if context cancels. Better to create new context in real app.
			_ = s.AutomationService.ExecuteFromTrigger(context.Background(), moduleName, validatedData, "create")

			// Webhook
			s.WebhookService.Trigger(context.Background(), "record.updated", models.WebhookPayload{
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

	// Apply Field Permissions
	perms, err := s.RoleService.GetFieldPermissions(ctx, userID, moduleName)
	if err == nil && perms != nil {
		for field, p := range perms {
			if p == models.FieldPermNone {
				delete(record, field)
			}
		}
	}

	return record, nil
}

func (s *RecordServiceImpl) ListRecords(ctx context.Context, moduleName string, filters map[string]any, page, limit int64, sortBy string, sortOrder string, userID primitive.ObjectID) ([]map[string]any, int64, error) {
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

	// Apply Field Permissions to all records
	perms, err := s.RoleService.GetFieldPermissions(ctx, userID, moduleName)
	if err == nil && perms != nil {
		for _, record := range records {
			for field, p := range perms {
				if p == models.FieldPermNone {
					delete(record, field)
				}
			}
		}
	}

	return records, totalCount, nil
}

func (s *RecordServiceImpl) UpdateRecord(ctx context.Context, moduleName, id string, data map[string]interface{}, userID primitive.ObjectID) error {
	// 1. Fetch Schema
	module, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return errors.New("module not found")
	}

	// 2. Validate Data (Partial update)
	validatedData := make(map[string]interface{})
	validatedData["updated_at"] = time.Now()

	// Allow owner to be updated if provided
	if ownerVal, exists := data["owner"]; exists {
		// Validate owner is a valid ObjectID
		if ownerStr, ok := ownerVal.(string); ok {
			if ownerID, err := primitive.ObjectIDFromHex(ownerStr); err == nil {
				validatedData["owner"] = ownerID
			}
		}
	}

	// Fetch Field Permissions
	perms, _ := s.RoleService.GetFieldPermissions(ctx, userID, moduleName)

	for _, field := range module.Fields {
		val, exists := data[field.Name]
		if !exists {
			continue // Partial update, skip missing fields
		}

		// Validate Type if present
		// Check Field Permissions
		if perms != nil {
			if p, ok := perms[field.Name]; ok {
				if p == models.FieldPermReadOnly || p == models.FieldPermNone {
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

		// 5. Automation & Webhook Trigger
		go func() {
			// Need the full updated record for conditions
			mergedRecord := make(map[string]interface{})
			for k, v := range oldRecord {
				mergedRecord[k] = v
			}
			for k, v := range validatedData {
				mergedRecord[k] = v
			}

			// Automation
			_ = s.AutomationService.ExecuteFromTrigger(context.Background(), moduleName, mergedRecord, "update")

			// Webhook
			s.WebhookService.Trigger(context.Background(), "record.updated", models.WebhookPayload{
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
		if field.Type == models.FieldTypeFile || field.Type == models.FieldTypeImage {
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
					refRecord, err := s.RecordRepo.Get(ctx, field.Lookup.LookupModule, idStr)
					if err == nil {
						// Determine Display Field
						displayField := "name" // Default
						if field.Lookup.LookupLabel != "" {
							displayField = field.Lookup.LookupLabel
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
	// Handle nil/empty
	if val == nil {
		return nil, nil
	}
	// Common empty check for "empty string" which means "null" for non-text fields
	if strVal, ok := val.(string); ok && strVal == "" {
		if field.Type != models.FieldTypeText && field.Type != models.FieldTypeTextArea && field.Type != models.FieldTypeSelect && field.Type != models.FieldTypeMultiSelect {
			return nil, nil
		}
	}

	switch field.Type {
	case models.FieldTypeNumber:
		switch v := val.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case string:
			if v == "" {
				return nil, nil // Should conform to check above, but redundant safety
			}
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, errors.New("expected number")
			}
			return f, nil
		default:
			return nil, errors.New("expected number")
		}
	case models.FieldTypeBoolean:
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
	case models.FieldTypeDate:
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
	case models.FieldTypeEmail:
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
	case models.FieldTypeLookup:
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

		// Integrity Check
		if field.Lookup != nil && field.Lookup.LookupModule != "" {
			// Check if referenced record exists
			_, err := s.RecordRepo.Get(ctx, field.Lookup.LookupModule, strVal)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					return nil, fmt.Errorf("referenced record in module '%s' not found", field.Lookup.LookupModule)
				}
				return nil, fmt.Errorf("failed to verify lookup reference: %v", err)
			}
		}

		return oid, nil
	case models.FieldTypeFile:
		strVal, ok := val.(string)
		if !ok {
			return nil, errors.New("expected string for file")
		}
		if strVal == "" {
			return nil, nil
		}

		// Check if it's a valid ObjectID (Reference)
		if _, err := primitive.ObjectIDFromHex(strVal); err == nil {
			// It's an ID, check existence in DB
			_, err = s.FileRepo.Get(ctx, strVal)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					return nil, errors.New("referenced file not found")
				}
				// If DB error, we can't verify, but maybe it's risky to fail?
				// Let's fail on checking error to be safe.
				return nil, fmt.Errorf("failed to verify file reference: %v", err)
			}
			// Return as is (ID string) or OID? Schema expects string usually for flexibility or OID?
			// Existing logic returned strVal.
			return strVal, nil
		}

		// If not ObjectID, assume it's a URL/Path (e.g. /uploads/...)
		// We could validate it starts with /uploads or http, but let's be permissive strictly for now
		return strVal, nil // Return the URL string

	case models.FieldTypeImage:
		strVal, ok := val.(string)
		if !ok {
			return nil, errors.New("expected string for image")
		}
		// Image is treated as string (URL or ID)
		return strVal, nil
	default:
		return val, nil
	}
}

func (s *RecordServiceImpl) getFieldPermissions(ctx context.Context, userID primitive.ObjectID, moduleName string) (map[string]string, error) {
	// 1. Get User
	user, err := s.UserRepo.FindByID(ctx, userID.Hex())
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// 2. Iterate Roles
	finalPerms := make(map[string]string)
	hasFieldRules := false

	for _, roleID := range user.Roles {
		role, err := s.RoleRepo.FindByID(ctx, roleID.Hex())
		if err != nil || role == nil {
			continue
		}

		if role.FieldPermissions != nil {
			if modPerms, ok := role.FieldPermissions[moduleName]; ok {
				hasFieldRules = true
				for field, p := range modPerms {
					current, exists := finalPerms[field]
					if !exists {
						finalPerms[field] = p
					} else {
						// Union: read_write > read_only > none
						// If we already have a permission, we take the MORE permissive one usually?
						// Wait, current logic:
						// if p == models.FieldPermReadWrite { finalPerms[field] = models.FieldPermReadWrite }
						// if p == models.FieldPermReadOnly && current == models.FieldPermNone { finalPerms[field] = models.FieldPermReadOnly }

						// Let's stick to simple "Least Restrictive"
						switch p {
						case models.FieldPermReadWrite:
							finalPerms[field] = models.FieldPermReadWrite
						case models.FieldPermReadOnly:
							if current == models.FieldPermNone {
								finalPerms[field] = models.FieldPermReadOnly
							}
						}
					}
				}
			}
		}

		if role.FieldPermissions == nil || role.FieldPermissions[moduleName] == nil {
			// This role grants full access to this module's fields.
			// Return nil effectively.
			return nil, nil
		}
	}

	if !hasFieldRules {
		return nil, nil
	}

	return finalPerms, nil
}
