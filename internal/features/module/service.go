package module

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/role"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ModuleService interface {
	CreateModule(ctx context.Context, module *Module) error
	GetModuleByName(ctx context.Context, name string, userID primitive.ObjectID) (*Module, error)
	ListModules(ctx context.Context, userID primitive.ObjectID, product string) ([]Module, error)
	UpdateModule(ctx context.Context, module *Module, userID primitive.ObjectID) error
	DeleteModule(ctx context.Context, name string, userID primitive.ObjectID) error
}

type ModuleServiceImpl struct {
	Repo         ModuleRepository
	RoleService  role.RoleService
	AuditService audit.AuditService
}

func NewModuleService(repo ModuleRepository, roleService role.RoleService, auditService audit.AuditService) ModuleService {
	return &ModuleServiceImpl{
		Repo:         repo,
		RoleService:  roleService,
		AuditService: auditService,
	}
}

func (s *ModuleServiceImpl) CreateModule(ctx context.Context, m *Module) error {
	// Basic Validation
	if m.Name == "" || m.Label == "" {
		return errors.New("module name and label are required")
	}

	// Check if already exists
	if _, err := s.Repo.FindByName(ctx, m.Name); err == nil {
		return errors.New("module with this name already exists")
	}

	m.ID = primitive.NewObjectID()
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()

	err := s.Repo.Create(ctx, m)
	if err == nil {
		changes := map[string]common_models.Change{
			"name":  {New: m.Name},
			"label": {New: m.Label},
		}
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionCreate, "module", m.ID.Hex(), changes)
	}
	return err
}

func (s *ModuleServiceImpl) GetModuleByName(ctx context.Context, name string, userID primitive.ObjectID) (*Module, error) {
	m, err := s.Repo.FindByName(ctx, name)
	if err != nil {
		return nil, err
	}
	s.appendSystemFields(m)

	// Permission Check
	if !userID.IsZero() {
		// 1. Global Read
		allowedGlobal, err := s.RoleService.CheckPermission(ctx, userID, "modules", "read")
		if err != nil || !allowedGlobal {
			// 2. Specific Read
			resourceID := fmt.Sprintf("%s.%s", m.Product, m.Name)
			allowedSpecific, errSpec := s.RoleService.CheckPermission(ctx, userID, resourceID, "read")
			if errSpec != nil || !allowedSpecific {
				return nil, errors.New("access denied")
			}
		}

		// Filter Fields based on FLS
		perms, _ := s.RoleService.GetFieldPermissions(ctx, userID, name)
		if perms != nil {
			visibleFields := []ModuleField{}
			for _, f := range m.Fields {
				if p, ok := perms[f.Name]; ok {
					if p == role.FieldPermNone {
						continue // Skip hidden fields
					}
				}
				visibleFields = append(visibleFields, f)
			}
			m.Fields = visibleFields
		}
	}

	return m, nil
}

func (s *ModuleServiceImpl) ListModules(ctx context.Context, userID primitive.ObjectID, product string) ([]Module, error) {
	modules, err := s.Repo.List(ctx, product)
	if err != nil {
		return nil, err
	}
	// Check for global "modules" read permission
	canReadAll := false
	if !userID.IsZero() {
		allowed, err := s.RoleService.CheckPermission(ctx, userID, "modules", "read")
		if err == nil && allowed {
			canReadAll = true
		}
	}

	filteredModules := make([]Module, 0, len(modules))
	for i := range modules {
		// Module Level Permission Check
		if !userID.IsZero() && !canReadAll {
			// Construct resource ID: e.g. "crm.leads"
			resourceID := fmt.Sprintf("%s.%s", modules[i].Product, modules[i].Name)
			allowed, err := s.RoleService.CheckPermission(ctx, userID, resourceID, "read")
			if err != nil || !allowed {
				continue // Skip module if no permission
			}
		}

		s.appendSystemFields(&modules[i])

		// Filter Fields (FLS)
		if !userID.IsZero() {
			perms, _ := s.RoleService.GetFieldPermissions(ctx, userID, modules[i].Name)
			if perms != nil {
				visibleFields := []ModuleField{}
				for _, f := range modules[i].Fields {
					if p, ok := perms[f.Name]; ok {
						if p == role.FieldPermNone {
							continue
						}
					}
					visibleFields = append(visibleFields, f)
				}
				modules[i].Fields = visibleFields
			}
		}
		filteredModules = append(filteredModules, modules[i])
	}
	return filteredModules, nil
}

func (s *ModuleServiceImpl) appendSystemFields(m *Module) {
	// Add Virtual System Fields
	systemFields := []ModuleField{
		{
			Name:     "created_at",
			Label:    "Created At",
			Type:     FieldTypeDate,
			Required: false,
			IsSystem: true,
		},
		{
			Name:     "updated_at",
			Label:    "Updated At",
			Type:     FieldTypeDate,
			Required: false,
			IsSystem: true,
		},
	}
	m.Fields = append(m.Fields, systemFields...)
}

func (s *ModuleServiceImpl) UpdateModule(ctx context.Context, m *Module, userID primitive.ObjectID) error {
	// Fetch existing module to compare fields
	existingModule, err := s.Repo.FindByName(ctx, m.Name)
	if err != nil {
		return err
	}

	// Permission Check
	if !userID.IsZero() {
		allowedGlobal, err := s.RoleService.CheckPermission(ctx, userID, "modules", "update")
		if err != nil || !allowedGlobal {
			resourceID := fmt.Sprintf("%s.%s", existingModule.Product, existingModule.Name)
			allowedSpecific, errSpec := s.RoleService.CheckPermission(ctx, userID, resourceID, "update")
			if errSpec != nil || !allowedSpecific {
				return errors.New("access denied")
			}
		}
	}

	// Identify removed fields
	existingFieldsMap := make(map[string]ModuleField)
	for _, f := range existingModule.Fields {
		existingFieldsMap[f.Name] = f
	}

	newFieldsMap := make(map[string]bool)
	for _, f := range m.Fields {
		newFieldsMap[f.Name] = true
	}

	var removedFields []string
	for name := range existingFieldsMap {
		if !newFieldsMap[name] {
			removedFields = append(removedFields, name)
		}
	}

	// Check if any removed field is used as display_field in other modules
	if len(removedFields) > 0 {
		// Find all modules that lookup TO this module
		dependentModules, err := s.Repo.FindUsingLookup(ctx, m.Name)
		if err != nil {
			return err
		}

		for _, depMod := range dependentModules {
			for _, f := range depMod.Fields {
				if f.Type == "lookup" && f.Lookup != nil && f.Lookup.LookupModule == m.Name {
					// Check if the display_field in the dependent module matches a removed field
					for _, removed := range removedFields {
						if f.Lookup.LookupLabel == removed {
							return fmt.Errorf("cannot remove field '%s', it is used as display field in module '%s' (field: '%s')", removed, depMod.Name, f.Name)
						}
					}
				}
			}
		}
	}

	// Preserve existing metadata if not provided in update
	if m.Label == "" {
		m.Label = existingModule.Label
	}
	m.ID = existingModule.ID
	m.IsSystem = existingModule.IsSystem
	m.CreatedAt = existingModule.CreatedAt
	m.UpdatedAt = time.Now()
	// In real app, we might check if module exists first or validate schema changes
	err = s.Repo.Update(ctx, m)
	if err == nil {
		// Log changes - ideally we calculate diff, but for now generic update
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionUpdate, "module", m.ID.Hex(), nil)
	}
	return err
}

func (s *ModuleServiceImpl) DeleteModule(ctx context.Context, name string, userID primitive.ObjectID) error {
	// 1. Check if System Module
	m, err := s.Repo.FindByName(ctx, name)
	if err != nil {
		return err
	}

	// Permission Check
	if !userID.IsZero() {
		allowedGlobal, err := s.RoleService.CheckPermission(ctx, userID, "modules", "delete")
		if err != nil || !allowedGlobal {
			resourceID := fmt.Sprintf("%s.%s", m.Product, m.Name)
			allowedSpecific, errSpec := s.RoleService.CheckPermission(ctx, userID, resourceID, "delete")
			if errSpec != nil || !allowedSpecific {
				return errors.New("access denied")
			}
		}
	}

	if m.IsSystem {
		return errors.New("cannot delete system module")
	}

	// 2. Dependency Check
	dependentModules, err := s.Repo.FindUsingLookup(ctx, name)
	if err != nil {
		return err
	}

	if len(dependentModules) > 0 {
		var depNames []string
		for _, m := range dependentModules {
			depNames = append(depNames, m.Name)
		}
		return fmt.Errorf("cannot delete module '%s', it is referenced by: %s module", name, strings.Join(depNames, ", "))
	}

	// 2. Data Cleanup now handled by Repo.Delete
	// if err := s.Repo.DropCollection(ctx, name); err != nil {
	// 	return fmt.Errorf("failed to drop module data: %w", err)
	// }

	// 3. Delete Metadata
	err = s.Repo.Delete(ctx, name)
	if err == nil {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionDelete, "module", name, nil)
	}
	return err
}
