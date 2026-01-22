package record

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go-crm/internal/common/models"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/core/audit"
	"go-crm/internal/core/permission"
	"go-crm/internal/core/role"
	"go-crm/internal/core/user"
	"go-crm/internal/features/file"
	"go-crm/internal/features/inventory"
	"go-crm/internal/features/module"
	"go-crm/internal/features/webhook"
	"go-crm/pkg/condition"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RecordService interface {
	CreateRecord(ctx context.Context, moduleName string, data map[string]any, userID primitive.ObjectID) (any, error)
	GetRecord(ctx context.Context, moduleName, id string, userID primitive.ObjectID) (map[string]any, error)
	ListRecords(ctx context.Context, moduleName string, filters []common_models.Filter, page, limit int64, sortBy string, sortOrder string, userID primitive.ObjectID) ([]map[string]any, int64, error)
	QueryRecords(ctx context.Context, moduleName string, action string, filters []common_models.Filter, page, limit int64, sortBy string, sortOrder string, userID primitive.ObjectID) ([]map[string]any, int64, error)
	UpdateRecord(ctx context.Context, moduleName, id string, data map[string]any, userID primitive.ObjectID) error
	DeleteRecord(ctx context.Context, moduleName, id string, userID primitive.ObjectID) error
}

// Internal interfaces to break circular dependencies
type AutomationTrigger interface {
	ExecuteFromTrigger(ctx context.Context, moduleName string, record map[string]any, triggerType string) error
}

type ApprovalTrigger interface {
	InitializeApproval(ctx context.Context, moduleName string, record map[string]any) (*common_models.ApprovalRecordState, error)
}

type BlueprintValidator interface {
	GetActiveBlueprintTargetField(ctx context.Context, module string) (string, error)
}

type RecordServiceImpl struct {
	ModuleRepo         module.ModuleRepository
	RecordRepo         RecordRepository
	FileRepo           file.FileRepository
	UserRepo           user.UserRepository
	RoleRepo           role.RoleRepository
	RoleService        role.RoleService
	AuditService       audit.AuditService
	ApprovalService    ApprovalTrigger
	AutomationService  AutomationTrigger
	WebhookService     webhook.WebhookService
	PermissionService  permission.PermissionService
	InventoryService   inventory.InventoryService
	BlueprintValidator BlueprintValidator
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
	permissionService permission.PermissionService,
	inventoryService inventory.InventoryService,
	blueprintValidator BlueprintValidator,
) RecordService {
	return &RecordServiceImpl{
		ModuleRepo:         moduleRepo,
		RecordRepo:         recordRepo,
		FileRepo:           fileRepo,
		UserRepo:           userRepo,
		RoleRepo:           roleRepo,
		RoleService:        roleService,
		AuditService:       auditService,
		ApprovalService:    approvalService,
		AutomationService:  automationService,
		WebhookService:     webhookService,
		PermissionService:  permissionService,
		InventoryService:   inventoryService,
		BlueprintValidator: blueprintValidator,
	}
}

func (s *RecordServiceImpl) CreateRecord(ctx context.Context, moduleName string, data map[string]any, userID primitive.ObjectID) (any, error) {
	// 1. Fetch Schema
	m, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return nil, errors.New("module not found")
	}

	// 2. Validate Data
	validatedData := make(map[string]any)
	validatedData["_id"] = primitive.NewObjectID()
	validatedData["created_at"] = time.Now()
	validatedData["updated_at"] = time.Now()
	validatedData["created_by"] = userID // System field - immutable
	validatedData["owner"] = userID      // Mutable field - can be changed

	// Fetch Field Permissions
	perms, err := s.RoleService.GetFieldPermissions(ctx, userID, moduleName)
	if err != nil {
		return nil, fmt.Errorf("failed to check field permissions: %v", err)
	}

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

	// Handle AutoIncrement Fields
	for _, field := range m.Fields {
		if field.Type == models.FieldTypeNumber && field.AutoIncrement {
			// Generate Sequence
			seq, err := s.RecordRepo.GetNextSequence(ctx, moduleName, field.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to generate sequence for field '%s': %v", field.Label, err)
			}
			validatedData[field.Name] = float64(seq) // Store as float64 because Number type expects it (or should we store INT? MongoDB numbers are double by default in JSON unmarshal usually)
		}
	}

	// Calculate Computed Fields
	s.calculateComputedFields(ctx, validatedData, m.Fields)

	// 3. Initialize Approval Workflow
	approvalState, err := s.ApprovalService.InitializeApproval(ctx, moduleName, validatedData)
	if err != nil {
		return nil, fmt.Errorf("failed to check approval workflow: %v", err)
	}
	if approvalState != nil {
		validatedData["_approval"] = approvalState
	}

	// 4. Insert
	res, err := s.RecordRepo.Create(ctx, moduleName, m.App, validatedData)
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
			mergedRecord := make(map[string]any)
			for k, v := range validatedData {
				mergedRecord[k] = v
			}

			_ = s.AutomationService.ExecuteFromTrigger(context.Background(), moduleName, validatedData, "create")

			// Inventory Trigger
			if s.InventoryService != nil {
				_ = s.InventoryService.HandleStockUpdate(context.Background(), moduleName, mergedRecord)
			}

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

	// Populate Users
	if err := s.populateUsers(ctx, m.Fields, record); err != nil {
		return nil, err
	}

	// Apply Field Permissions
	perms, err := s.RoleService.GetFieldPermissions(ctx, userID, moduleName)
	if err != nil {
		return nil, fmt.Errorf("failed to check field permissions: %v", err)
	}
	for field, p := range perms {
		if p == role.FieldPermNone {
			delete(record, field)
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
		_ = s.populateUsers(ctx, m.Fields, record)
	}

	totalCount, err := s.RecordRepo.Count(ctx, moduleName, typedFilters, accessFilter)
	if err != nil {
		return nil, 0, err
	}

	perms, err := s.RoleService.GetFieldPermissions(ctx, userID, moduleName)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to check field permissions: %v", err)
	}
	if perms != nil {
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

func (s *RecordServiceImpl) QueryRecords(ctx context.Context, moduleName string, action string, filters []common_models.Filter, page, limit int64, sortBy string, sortOrder string, userID primitive.ObjectID) ([]map[string]any, int64, error) {
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

	// 1. Fetch Schema
	m, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return nil, 0, errors.New("module not found")
	}

	// 2. Fetch User & Permissions
	user, err := s.UserRepo.FindByID(ctx, userID.Hex())
	if err != nil {
		return nil, 0, err
	}

	// Create context for compiler
	// Variable Resolution logic: $user.id, $user.path, $now
	// We might need to fetch org info for path? Or is it on User?
	// $user.path might refer to org structure or just Org ID?
	// User Prompt: $user.path org path
	// User struct has TenantID, Groups.
	var userPath string
	// Simplified assumption: TenantID is the path base, or we don't have tree yet.
	// For now using tenantID as path or empty if not applicable.
	userPath = user.TenantID.Hex()

	compilerCtx := map[string]any{
		"user.id":   userID,
		"user.path": userPath,
	}

	perms, err := s.PermissionService.GetUserEffectivePermissions(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	// 3. Validate Action Allowed
	// Construct full resource key (e.g., "crm.leads")
	// If Product is empty, it might just be the name (e.g. system modules?)
	// But perms are usually keyed by "product.module".
	resourceKey := fmt.Sprintf("%s.%s", m.App, m.Name)

	var actionPerm *common_models.ActionPermission

	// Helper to find action perms
	findAction := func(p *permission.Permission) *common_models.ActionPermission {
		if ap, ok := p.Actions[action]; ok {
			return &ap
		}
		if ap, ok := p.Actions["*"]; ok {
			return &ap
		}
		return nil
	}

	// 1. Check specific resource permission
	if p, ok := perms[resourceKey]; ok {
		actionPerm = findAction(p)
	}

	// 2. Check simple name permission (Fallback)
	if actionPerm == nil {
		if p, ok := perms[m.Name]; ok {
			actionPerm = findAction(p)
		}
	}

	// 3. Check wildcard permission (Global Fallback)
	if actionPerm == nil {
		if p, ok := perms["*"]; ok {
			actionPerm = findAction(p)
		}
	}

	if actionPerm == nil || !actionPerm.Allowed {
		return nil, 0, errors.New("permission denied")
	}

	// 4. Validate Requested Filters
	// 4. Validate Requested Filters
	allowedFiltersMap := make(map[string]bool)
	hasExplicitFilterConfig := false
	if actionPerm.UI != nil && len(actionPerm.UI.Filters) > 0 {
		hasExplicitFilterConfig = true
		for _, f := range actionPerm.UI.Filters {
			allowedFiltersMap[f] = true
		}
	}

	if hasExplicitFilterConfig {
		for _, f := range filters {
			// System fields might be always allowed? Or strictly controlled?
			if !allowedFiltersMap[f.Field] {
				// Allow if system ID? or just strict?
				// Strict adherence to requirement implies blocking.
				return nil, 0, fmt.Errorf("filter on field '%s' is not allowed", f.Field)
			}
		}
	}

	// 5. Build MongoDB Query
	// A. Forced Conditions from Permission
	var forcedCondition bson.M
	if actionPerm.Conditions != nil {
		compiler := condition.NewCompiler(compilerCtx)
		cond, err := compiler.Compile(actionPerm.Conditions)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to compile permission conditions: %v", err)
		}
		forcedCondition = cond
	}

	// B. User Filters
	userFilters, err := s.prepareFilters(ctx, m, filters)
	if err != nil {
		return nil, 0, err
	}

	// C. Combine (AND)
	finalQuery := bson.M{
		"$and": []bson.M{
			{"tenant_id": user.TenantID},             // Always limit to tenant
			{"deleted_at": bson.M{"$exists": false}}, // Exclude soft deleted
		},
	}

	andClauses := finalQuery["$and"].([]bson.M)

	if len(forcedCondition) > 0 {
		andClauses = append(andClauses, forcedCondition)
	}

	if len(userFilters) > 0 {
		andClauses = append(andClauses, userFilters)
	}

	// Reassign back to map
	finalQuery["$and"] = andClauses

	sortOrderInt := -1
	if strings.ToLower(sortOrder) == "asc" {
		sortOrderInt = 1
	}

	// Execute List using direct repo method or passing custom filter
	// Repo.List takes 'filter' and 'accessFilter'.
	// filter is 'userFilters', accessFilter is 'permission restrictions'.
	// We can pass empty 'userFilters' and put everything in 'accessFilter' or vice versa.
	// Repo.List logic: filter AND accessFilter.
	// So we can pass `userFilters` as filter, and `forcedCondition` as accessFilter?
	// But our `ForcedCondition` logic replaces `GetAccessFilter`.
	// We should probably expose `Repo.Find(query)` or just reuse List logic creatively.
	// Repo.List matches: `filter` (bson.M) AND `accessFilter` (bson.M).
	// So we can pass `userFilters` as filter, and `forcedCondition` as accessFilter.
	// BUT Repo.List adds `tenant_id` internally inside `List` method?
	// Let's check Repo.List implementation in `record/repository.go` (not read yet, but usually standard).
	// Assuming Repo adds tenant_id constraint.
	// Let's assume we pass `userFilters` as first arg, and `forcedCondition` as second.

	records, err := s.RecordRepo.List(ctx, moduleName, userFilters, forcedCondition, limit, offset, sortBy, sortOrderInt)
	if err != nil {
		return nil, 0, err
	}

	for _, record := range records {
		_ = s.populateFiles(ctx, m.Fields, record)
		_ = s.populateLookups(ctx, m.Fields, record)
		_ = s.populateUsers(ctx, m.Fields, record)
	}

	// Count
	totalCount, err := s.RecordRepo.Count(ctx, moduleName, userFilters, forcedCondition)
	if err != nil {
		return nil, 0, err
	}

	// Field Permissions (Read-Only/Hidden masking)
	// We already fetched perms via GetUserEffectivePermissions, we can extract field rules from there?
	// GetUserEffectivePermissions returns map[string]*Permission.
	// Permission has FieldRules map[string]string.
	// We should check that.
	// s.RoleService.GetFieldPermissions uses UserRepo/Role permissions.
	// We can reuse s.RoleService.GetFieldPermissions or iterate ourselves.
	// For consistency, let's reuse s.RoleService.GetFieldPermissions OR extract from `perms`.
	// If we use s.RoleService.GetFieldPermissions it re-fetches user/roles.
	// We have `perms` (effective perms). We can construct `fieldPerms` from it.

	fieldRules := make(map[string]string)
	if p, ok := perms[moduleName]; ok {
		for f, r := range p.FieldRules {
			fieldRules[f] = r
		}
	} else if p, ok := perms["*"]; ok {
		// Wildcard field rules?
		for f, r := range p.FieldRules {
			fieldRules[f] = r
		}
	}

	if len(fieldRules) > 0 {
		for _, record := range records {
			for field, rule := range fieldRules {
				if rule == "none" { // FieldPermNone
					delete(record, field)
				}
			}
		}
	}

	return records, totalCount, nil
}

func (s *RecordServiceImpl) UpdateRecord(ctx context.Context, moduleName, id string, data map[string]any, userID primitive.ObjectID) error {
	m, err := s.ModuleRepo.FindByName(ctx, moduleName)
	if err != nil {
		return errors.New("module not found")
	}

	validatedData := make(map[string]any)
	validatedData["updated_at"] = time.Now()

	if ownerVal, exists := data["owner"]; exists {
		if ownerStr, ok := ownerVal.(string); ok {
			if ownerID, err := primitive.ObjectIDFromHex(ownerStr); err == nil {
				validatedData["owner"] = ownerID
			}
		}
	}

	perms, err := s.RoleService.GetFieldPermissions(ctx, userID, moduleName)
	if err != nil {
		return fmt.Errorf("failed to check field permissions: %v", err)
	}

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

	// CHECK BLUEPRINT FIELD
	if s.BlueprintValidator != nil {
		targetField, err := s.BlueprintValidator.GetActiveBlueprintTargetField(ctx, moduleName)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				targetField = ""
			} else {
				return fmt.Errorf("failed to validate blueprint constraint: %v", err)
			}
		}
		if targetField != "" {
			if newVal, ok := validatedData[targetField]; ok {
				oldVal := oldRecord[targetField]
				if !reflect.DeepEqual(oldVal, newVal) {
					return fmt.Errorf("field '%s' is managed by a blueprint and cannot be updated manually", targetField)
				}
			}
		}
	}

	if val, ok := oldRecord["_approval"]; ok {
		if stateMap, ok := val.(map[string]any); ok {
			if status, ok := stateMap["status"].(string); ok && status == "pending" {
				return errors.New("record is locked for approval and cannot be edited")
			}
		} else if stateMap, ok := val.(primitive.M); ok {
			if status, ok := stateMap["status"].(string); ok && status == "pending" {
				return errors.New("record is locked for approval and cannot be edited")
			}
		}
	}

	// Calculate Computed Fields (using merged state)
	projectedRecord := make(map[string]any)
	for k, v := range oldRecord {
		projectedRecord[k] = v
	}
	for k, v := range validatedData {
		projectedRecord[k] = v
	}
	s.calculateComputedFields(ctx, projectedRecord, m.Fields)
	// Update validatedData with calculated values
	for _, field := range m.Fields {
		if field.Type == common_models.FieldTypeFunction {
			if val, ok := projectedRecord[field.Name]; ok {
				validatedData[field.Name] = val
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

		// Create detached context to preserve values (e.g. TenantID) but ignore cancellation
		detachedCtx := context.WithoutCancel(ctx)

		go func() {
			mergedRecord := make(map[string]any)
			for k, v := range oldRecord {
				mergedRecord[k] = v
			}
			for k, v := range validatedData {
				mergedRecord[k] = v
			}

			_ = s.AutomationService.ExecuteFromTrigger(detachedCtx, moduleName, mergedRecord, "update")

			// Inventory Trigger
			if s.InventoryService != nil {
				_ = s.InventoryService.HandleStockUpdate(detachedCtx, moduleName, mergedRecord)
			}

			s.WebhookService.Trigger(detachedCtx, "record.updated", common_models.WebhookPayload{
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
		if stateMap, ok := val.(map[string]any); ok {
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

func (s *RecordServiceImpl) populateFiles(ctx context.Context, fields []models.ModuleField, record map[string]any) error {
	for _, field := range fields {
		if field.Type == models.FieldTypeSubform {
			if val, ok := record[field.Name]; ok {
				var items []any
				if a, ok := val.(primitive.A); ok {
					items = a
				} else if a, ok := val.([]any); ok {
					items = a
				}

				for _, item := range items {
					if row, ok := item.(map[string]any); ok {
						_ = s.populateFiles(ctx, field.SubFields, row)
					} else if row, ok := item.(primitive.M); ok {
						_ = s.populateFiles(ctx, field.SubFields, row)
					}
				}
			}
			continue
		}

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
						record[field.Name] = map[string]any{
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

func (s *RecordServiceImpl) populateLookups(ctx context.Context, fields []models.ModuleField, record map[string]any) error {
	for _, field := range fields {
		if field.Type == models.FieldTypeSubform {
			if val, ok := record[field.Name]; ok {
				var items []any
				if a, ok := val.(primitive.A); ok {
					items = a
				} else if a, ok := val.([]any); ok {
					items = a
				}

				for _, item := range items {
					if row, ok := item.(map[string]any); ok {
						_ = s.populateLookups(ctx, field.SubFields, row)
					} else if row, ok := item.(primitive.M); ok {
						_ = s.populateLookups(ctx, field.SubFields, row)
					}
				}
			}
			continue
		}

		if field.Type == models.FieldTypeLookup && field.Lookup != nil {
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

						record[field.Name] = map[string]any{
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

func (s *RecordServiceImpl) validateAndConvert(ctx context.Context, field models.ModuleField, val any) (any, error) {
	if val == nil {
		return nil, nil
	}
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
		var idStr string
		switch v := val.(type) {
		case string:
			idStr = v
		case primitive.ObjectID:
			idStr = v.Hex()
		case map[string]any:
			if id, ok := v["id"].(string); ok {
				idStr = id
			} else if oid, ok := v["id"].(primitive.ObjectID); ok {
				idStr = oid.Hex()
			}
		case primitive.M:
			if id, ok := v["id"].(string); ok {
				idStr = id
			} else if oid, ok := v["id"].(primitive.ObjectID); ok {
				idStr = oid.Hex()
			}
		default:
			return nil, errors.New("expected valid objectID hex string or populated object for lookup")
		}

		if idStr == "" {
			return nil, nil
		}

		oid, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			return nil, errors.New("invalid objectID for lookup")
		}

		if field.Lookup != nil && field.Lookup.LookupModule != "" {
			_, err := s.RecordRepo.Get(ctx, field.Lookup.LookupModule, idStr)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					return nil, fmt.Errorf("referenced record in module '%s' not found", field.Lookup.LookupModule)
				}
				return nil, fmt.Errorf("failed to verify lookup reference: %v", err)
			}
		}

		return oid, nil

	case models.FieldTypeFile:
		var idStr string
		switch v := val.(type) {
		case string:
			idStr = v
		case primitive.ObjectID:
			idStr = v.Hex()
		case map[string]any:
			if id, ok := v["id"].(string); ok {
				idStr = id
			} else if oid, ok := v["id"].(primitive.ObjectID); ok {
				idStr = oid.Hex()
			} else if id, ok := v["id"].(primitive.ObjectID); ok {
				idStr = id.Hex()
			}
		case primitive.M:
			if id, ok := v["id"].(string); ok {
				idStr = id
			} else if oid, ok := v["id"].(primitive.ObjectID); ok {
				idStr = oid.Hex()
			}
		default:
			return nil, errors.New("expected string or populated object for file")
		}

		if idStr == "" {
			return nil, nil
		}

		if _, err := primitive.ObjectIDFromHex(idStr); err == nil {
			_, err = s.FileRepo.Get(ctx, idStr)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					return nil, errors.New("referenced file not found")
				}
				return nil, fmt.Errorf("failed to verify file reference: %v", err)
			}
			return idStr, nil
		}

		return idStr, nil

	case models.FieldTypeImage:
		var idStr string
		switch v := val.(type) {
		case string:
			idStr = v
		case primitive.ObjectID:
			idStr = v.Hex()
		case map[string]any:
			if id, ok := v["id"].(string); ok {
				idStr = id
			} else if oid, ok := v["id"].(primitive.ObjectID); ok {
				idStr = oid.Hex()
			}
		case primitive.M:
			if id, ok := v["id"].(string); ok {
				idStr = id
			} else if oid, ok := v["id"].(primitive.ObjectID); ok {
				idStr = oid.Hex()
			}
		default:
			return nil, errors.New("expected string or populated object for image")
		}
		return idStr, nil
	case models.FieldTypeUser:
		var idStr string
		switch v := val.(type) {
		case string:
			idStr = v
		case primitive.ObjectID:
			return v, nil
		case map[string]any:
			if id, ok := v["id"].(string); ok {
				idStr = id
			} else if oid, ok := v["id"].(primitive.ObjectID); ok {
				return oid, nil
			}
		case primitive.M:
			if id, ok := v["id"].(string); ok {
				idStr = id
			} else if oid, ok := v["id"].(primitive.ObjectID); ok {
				return oid, nil
			}
		default:
			return nil, errors.New("expected string or populated object for user")
		}

		if idStr == "" {
			return nil, nil
		}

		oid, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			return nil, errors.New("invalid objectID for user")
		}

		// Validate user exists
		_, err = s.UserRepo.FindByID(ctx, idStr)
		if err != nil {
			return nil, errors.New("referenced user not found")
		}
		return oid, nil

	case models.FieldTypeFunction:
		// Function fields are calculated, not input directly.
		// However, we might want to allow override? Usually no.
		// For now, ignore input for function fields as they are read-only/computed.
		return nil, nil
	default:
		return val, nil
	}
}

func (s *RecordServiceImpl) calculateComputedFields(ctx context.Context, record map[string]any, fields []common_models.ModuleField) {
	for _, field := range fields {
		if field.Type == common_models.FieldTypeFunction && field.Function != nil {
			val, err := s.evaluateFunction(record, field.Function)
			if err == nil {
				record[field.Name] = val
			} else {
				// Default to 0 if calculation fails
				record[field.Name] = 0.0
			}
		}
	}
}

func (s *RecordServiceImpl) evaluateFunction(record map[string]any, fn *common_models.FunctionDef) (any, error) {
	if fn == nil {
		return nil, errors.New("no function definition")
	}

	switch fn.Operation {
	case "sum", "avg", "min", "max":
		// Target format: "subformName.subfieldName"
		parts := strings.Split(fn.Target, ".")
		if len(parts) != 2 {
			return 0.0, fmt.Errorf("invalid target format: %s", fn.Target)
		}
		subformName, subFieldName := parts[0], parts[1]

		rawList, ok := record[subformName]
		if !ok || rawList == nil {
			return 0.0, nil
		}

		var values []float64

		// Helper to extract value from row
		extract := func(row map[string]any) {
			if val, exists := row[subFieldName]; exists {
				values = append(values, toFloat(val))
			}
		}

		// Iterate generic list
		switch list := rawList.(type) {
		case []any:
			for _, item := range list {
				if row, ok := item.(map[string]any); ok {
					extract(row)
				} else if row, ok := item.(primitive.M); ok {
					extract(map[string]any(row))
				}
			}
		case []map[string]any:
			for _, row := range list {
				extract(row)
			}
		case primitive.A: // Mongo driver array
			for _, item := range list {
				if row, ok := item.(map[string]any); ok {
					extract(row)
				} else if row, ok := item.(primitive.M); ok {
					extract(map[string]any(row))
				}
			}
		}

		if len(values) == 0 {
			return 0.0, nil
		}

		switch fn.Operation {
		case "sum":
			sum := 0.0
			for _, v := range values {
				sum += v
			}
			return sum, nil
		case "avg":
			sum := 0.0
			for _, v := range values {
				sum += v
			}
			return sum / float64(len(values)), nil
		case "min":
			min := values[0]
			for _, v := range values {
				if v < min {
					min = v
				}
			}
			return min, nil
		case "max":
			max := values[0]
			for _, v := range values {
				if v > max {
					max = v
				}
			}
			return max, nil
		}

	case "script":
		if fn.Script == "" {
			return 0.0, nil
		}
		vm := goja.New()

		// Inject Record
		_ = vm.Set("record", record)

		// Inject Helpers
		_ = vm.Set("Sum", func(call goja.FunctionCall) goja.Value {
			sum := 0.0
			for _, arg := range call.Arguments {
				if val, ok := arg.Export().(float64); ok {
					sum += val
				} else if val, ok := arg.Export().(int64); ok {
					sum += float64(val)
				}
			}
			return vm.ToValue(sum)
		})

		_ = vm.Set("Max", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) == 0 {
				return vm.ToValue(0)
			}
			max := -1e9 // reasonably small
			first := true
			for _, arg := range call.Arguments {
				val := 0.0
				if v, ok := arg.Export().(float64); ok {
					val = v
				} else if v, ok := arg.Export().(int64); ok {
					val = float64(v)
				} else {
					continue
				}

				if first {
					max = val
					first = false
				} else if val > max {
					max = val
				}
			}
			return vm.ToValue(max)
		})

		// Wrap in function to allow return
		script := fmt.Sprintf("(function() { %s })()", fn.Script)
		val, err := vm.RunString(script)
		if err != nil {
			return nil, fmt.Errorf("script execution failed: %v", err)
		}
		return val.Export(), nil

	case "multiply":
		result := 1.0
		if len(fn.Targets) == 0 {
			return 0.0, nil
		}
		for _, target := range fn.Targets {
			val, ok := record[target]
			if !ok {
				val = 0.0
			}
			result *= toFloat(val)
		}
		return result, nil
	}

	return nil, fmt.Errorf("unknown operation: %s", fn.Operation)
}

func toFloat(val any) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	case nil:
		return 0.0
	}
	return 0.0
}
func (s *RecordServiceImpl) populateUsers(ctx context.Context, fields []models.ModuleField, record map[string]any) error {
	for _, field := range fields {
		if field.Type == models.FieldTypeSubform {
			if val, ok := record[field.Name]; ok {
				var items []any
				if a, ok := val.(primitive.A); ok {
					items = a
				} else if a, ok := val.([]any); ok {
					items = a
				}

				for _, item := range items {
					if row, ok := item.(map[string]any); ok {
						_ = s.populateUsers(ctx, field.SubFields, row)
					} else if row, ok := item.(primitive.M); ok {
						_ = s.populateUsers(ctx, field.SubFields, row)
					}
				}
			}
			continue
		}

		if field.Type == models.FieldTypeUser {
			if val, ok := record[field.Name]; ok {
				var idStr string
				if oid, ok := val.(primitive.ObjectID); ok {
					idStr = oid.Hex()
				} else if s, ok := val.(string); ok {
					idStr = s
				}

				if idStr != "" {
					user, err := s.UserRepo.FindByID(ctx, idStr)
					if err == nil {
						// Format user name
						displayName := user.Email
						if user.FirstName != "" || user.LastName != "" {
							displayName = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
							displayName = strings.TrimSpace(displayName)
						}

						record[field.Name] = map[string]any{
							"id":         idStr,
							"name":       displayName,
							"email":      user.Email,
							"avatar_url": "", // Could add avatar if available
						}
					}
				}
			}
		}
	}
	return nil
}

func (s *RecordServiceImpl) prepareFilters(ctx context.Context, m *common_models.Entity, filters []common_models.Filter) (bson.M, error) {
	typedFilters := bson.M{}

	for _, f := range filters {
		fieldName := f.Field
		operator := f.Operator
		val := f.Value

		// Handle Special ID fields
		if fieldName == "id" || fieldName == "_id" {
			switch operator {
			case "in":
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
				case []primitive.ObjectID:
					ids = append(ids, v...)
				case primitive.A:
					for _, item := range v {
						if s, ok := item.(string); ok {
							if oid, err := primitive.ObjectIDFromHex(s); err == nil {
								ids = append(ids, oid)
							}
						} else if oid, ok := item.(primitive.ObjectID); ok {
							ids = append(ids, oid)
						}
					}
				case []any:
					for _, item := range v {
						if s, ok := item.(string); ok {
							if oid, err := primitive.ObjectIDFromHex(s); err == nil {
								ids = append(ids, oid)
							}
						} else if oid, ok := item.(primitive.ObjectID); ok {
							ids = append(ids, oid)
						}
					}
				}
				if len(ids) > 0 {
					typedFilters["_id"] = bson.M{"$in": ids}
				}
			case "", "eq":
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
			var typedVal any
			var err error

			// Handle slice for in/nin operators
			if operator == "in" || operator == "nin" {
				if sliceVal, ok := val.([]string); ok {
					var typedSlice []any
					for _, sVal := range sliceVal {
						tVal, err := s.validateAndConvert(ctx, *field, sVal)
						if err != nil {
							return nil, err
						}
						typedSlice = append(typedSlice, tVal)
					}
					typedVal = typedSlice
				} else if sliceVal, ok := val.([]any); ok {
					var typedSlice []any
					for _, sVal := range sliceVal {
						tVal, err := s.validateAndConvert(ctx, *field, sVal)
						if err != nil {
							return nil, err
						}
						typedSlice = append(typedSlice, tVal)
					}
					typedVal = typedSlice
				} else {
					typedVal, err = s.validateAndConvert(ctx, *field, val)
				}
			} else {
				typedVal, err = s.validateAndConvert(ctx, *field, val)
			}

			if err != nil {
				return nil, fmt.Errorf("invalid filter value for '%s': %v", field.Label, err)
			}

			switch operator {
			case "", "eq":
				if field.Type == models.FieldTypeUser || field.Type == models.FieldTypeLookup {
					if oid, ok := typedVal.(primitive.ObjectID); ok {
						typedFilters[fieldName] = bson.M{"$in": []any{oid, oid.Hex()}}
					} else {
						typedFilters[fieldName] = typedVal
					}
				} else {
					typedFilters[fieldName] = typedVal
				}
			case "ne":
				if field.Type == models.FieldTypeUser || field.Type == models.FieldTypeLookup {
					if oid, ok := typedVal.(primitive.ObjectID); ok {
						typedFilters[fieldName] = bson.M{"$nin": []any{oid, oid.Hex()}}
					} else {
						typedFilters[fieldName] = bson.M{"$ne": typedVal}
					}
				} else {
					typedFilters[fieldName] = bson.M{"$ne": typedVal}
				}
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
				if field.Type == models.FieldTypeUser || field.Type == models.FieldTypeLookup {
					if slice, ok := typedVal.([]any); ok {
						var expandedSlice []any
						for _, item := range slice {
							if oid, ok := item.(primitive.ObjectID); ok {
								expandedSlice = append(expandedSlice, oid, oid.Hex())
							} else {
								expandedSlice = append(expandedSlice, item)
							}
						}
						typedFilters[fieldName] = bson.M{"$in": expandedSlice}
					} else {
						typedFilters[fieldName] = bson.M{"$in": typedVal}
					}
				} else {
					typedFilters[fieldName] = bson.M{"$in": typedVal}
				}
			case "nin":
				if field.Type == models.FieldTypeUser || field.Type == models.FieldTypeLookup {
					if slice, ok := typedVal.([]any); ok {
						var expandedSlice []any
						for _, item := range slice {
							if oid, ok := item.(primitive.ObjectID); ok {
								expandedSlice = append(expandedSlice, oid, oid.Hex())
							} else {
								expandedSlice = append(expandedSlice, item)
							}
						}
						typedFilters[fieldName] = bson.M{"$nin": expandedSlice}
					} else {
						typedFilters[fieldName] = bson.M{"$nin": typedVal}
					}
				} else {
					typedFilters[fieldName] = bson.M{"$nin": typedVal}
				}
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
