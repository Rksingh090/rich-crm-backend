package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ModuleServiceImpl struct {
	Repo         repository.ModuleRepository
	AuditService AuditService
}

func NewModuleServiceImpl(repo repository.ModuleRepository, auditService AuditService) *ModuleServiceImpl {
	return &ModuleServiceImpl{
		Repo:         repo,
		AuditService: auditService,
	}
}

func (s *ModuleServiceImpl) CreateModule(ctx context.Context, module *models.Module) error {
	// Basic Validation
	if module.Name == "" || module.Label == "" {
		return errors.New("module name and label are required")
	}

	// Check if already exists
	if _, err := s.Repo.FindByName(ctx, module.Name); err == nil {
		return errors.New("module with this name already exists")
	}

	module.ID = primitive.NewObjectID()
	module.CreatedAt = time.Now()
	module.UpdatedAt = time.Now()

	err := s.Repo.Create(ctx, module)
	if err == nil {
		changes := map[string]models.Change{
			"name":  {New: module.Name},
			"label": {New: module.Label},
		}
		_ = s.AuditService.LogChange(ctx, models.AuditActionCreate, "module", module.ID.Hex(), changes)
	}
	return err
}

func (s *ModuleServiceImpl) GetModuleByName(ctx context.Context, name string) (*models.Module, error) {
	return s.Repo.FindByName(ctx, name)
}

func (s *ModuleServiceImpl) ListModules(ctx context.Context) ([]models.Module, error) {
	return s.Repo.List(ctx)
}

func (s *ModuleServiceImpl) UpdateModule(ctx context.Context, module *models.Module) error {
	// Fetch existing module to compare fields
	existingModule, err := s.Repo.FindByName(ctx, module.Name)
	if err != nil {
		return err
	}

	// Identify removed fields
	existingFieldsMap := make(map[string]models.ModuleField)
	for _, f := range existingModule.Fields {
		existingFieldsMap[f.Name] = f
	}

	newFieldsMap := make(map[string]bool)
	for _, f := range module.Fields {
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
		dependentModules, err := s.Repo.FindUsingLookup(ctx, module.Name)
		if err != nil {
			return err
		}

		for _, depMod := range dependentModules {
			for _, f := range depMod.Fields {
				if f.Type == "lookup" && f.Lookup != nil && f.Lookup.Module == module.Name {
					// Check if the display_field in the dependent module matches a removed field
					for _, removed := range removedFields {
						if f.Lookup.DisplayField == removed {
							return fmt.Errorf("cannot remove field '%s', it is used as display field in module '%s' (field: '%s')", removed, depMod.Name, f.Name)
						}
					}
				}
			}
		}
	}

	module.UpdatedAt = time.Now()
	// In real app, we might check if module exists first or validate schema changes
	err = s.Repo.Update(ctx, module)
	if err == nil {
		// Log changes - ideally we calculate diff, but for now generic update
		_ = s.AuditService.LogChange(ctx, models.AuditActionUpdate, "module", module.ID.Hex(), nil)
	}
	return err
}

func (s *ModuleServiceImpl) DeleteModule(ctx context.Context, name string) error {
	// 1. Dependency Check
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

	// 2. Data Cleanup (Drop Collection)
	// We assume collection name matches module name.
	// Note: Should wrap in transaction for full safety, but simple sequential ok for now.
	if err := s.Repo.DropCollection(ctx, name); err != nil {
		// Log error but proceed? Or fail?
		// If collection doesn't exist, mongo might return error or not.
		// For now, let's treat it as non-critical or check specific error.
		// A simple way is to proceed if it's just "ns not found", but let's just return error for safety.
		// Iterate: if we fail to drop data, we shouldn't delete metadata.
		return fmt.Errorf("failed to drop module data: %w", err)
	}

	// 3. Delete Metadata
	// 3. Delete Metadata
	err = s.Repo.Delete(ctx, name)
	if err == nil {
		_ = s.AuditService.LogChange(ctx, models.AuditActionDelete, "module", name, nil)
	}
	return err
}
