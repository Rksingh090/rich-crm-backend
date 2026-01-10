package resource

import (
	"context"
	"fmt"
	"time"

	"go-crm/internal/common/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ResourceService interface {
	// Register a new resource (called when tenant creates custom module)
	RegisterResource(ctx context.Context, resource *Resource) error

	// Get all resources for a tenant (includes global, app-level, and tenant-specific)
	GetResourcesForTenant(ctx context.Context, tenantID primitive.ObjectID, app models.App) ([]Resource, error)

	// Get a specific resource by ID
	GetResourceByID(ctx context.Context, resourceID string, tenantID *primitive.ObjectID) (*Resource, error)

	// Update resource metadata
	UpdateResource(ctx context.Context, id primitive.ObjectID, resource *Resource) error

	// Delete resource (soft delete for tenant resources, hard delete for system resources requires special permission)
	DeleteResource(ctx context.Context, id primitive.ObjectID, tenantID primitive.ObjectID) error

	// Discover resources by type
	DiscoverResourcesByType(ctx context.Context, tenantID primitive.ObjectID, app models.App, resourceType ResourceType) ([]Resource, error)

	// Get all global resources for an app
	GetGlobalResourcesForApp(ctx context.Context, app models.App) ([]Resource, error)
}

type ResourceServiceImpl struct {
	repo ResourceRepository
}

func NewResourceService(repo ResourceRepository) ResourceService {
	return &ResourceServiceImpl{
		repo: repo,
	}
}

func (s *ResourceServiceImpl) RegisterResource(ctx context.Context, resource *Resource) error {
	// Validate resource
	if resource.ResourceID == "" {
		return fmt.Errorf("resource_id is required")
	}

	if resource.App == "" {
		return fmt.Errorf("app is required")
	}

	if resource.Type == "" {
		return fmt.Errorf("type is required")
	}

	// Set defaults
	if resource.ID.IsZero() {
		resource.ID = primitive.NewObjectID()
	}

	if len(resource.AvailableActions) == 0 {
		// Default actions for most resources
		resource.AvailableActions = []string{"read", "create", "update", "delete"}
	}

	// Determine scope based on TenantID
	if resource.TenantID != nil {
		resource.Scope = ResourceScopeTenant
	} else if resource.Scope == "" {
		// Default to app-level if not specified
		resource.Scope = ResourceScopeApp
	}

	resource.CreatedAt = time.Now()
	resource.UpdatedAt = time.Now()

	return s.repo.Create(ctx, resource)
}

func (s *ResourceServiceImpl) GetResourcesForTenant(ctx context.Context, tenantID primitive.ObjectID, app models.App) ([]Resource, error) {
	filter := ResourceFilter{
		TenantID:      &tenantID,
		App:           &app,
		IncludeGlobal: true, // Include global and app-level resources
	}

	return s.repo.FindAll(ctx, filter)
}

func (s *ResourceServiceImpl) GetResourceByID(ctx context.Context, resourceID string, tenantID *primitive.ObjectID) (*Resource, error) {
	return s.repo.FindByID(ctx, resourceID, tenantID)
}

func (s *ResourceServiceImpl) UpdateResource(ctx context.Context, id primitive.ObjectID, resource *Resource) error {
	// Prevent updating system resources
	existing, err := s.repo.FindByResourceID(ctx, resource.ResourceID)
	if err != nil {
		return err
	}

	if existing.IsSystem {
		return fmt.Errorf("cannot update system resource")
	}

	resource.UpdatedAt = time.Now()
	return s.repo.Update(ctx, id, resource)
}

func (s *ResourceServiceImpl) DeleteResource(ctx context.Context, id primitive.ObjectID, tenantID primitive.ObjectID) error {
	// Find the resource first
	filter := ResourceFilter{
		TenantID: &tenantID,
	}

	resources, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return err
	}

	var targetResource *Resource
	for _, r := range resources {
		if r.ID == id {
			targetResource = &r
			break
		}
	}

	if targetResource == nil {
		return fmt.Errorf("resource not found")
	}

	// Prevent deleting system resources
	if targetResource.IsSystem {
		return fmt.Errorf("cannot delete system resource")
	}

	// Prevent deleting resources from other tenants
	if targetResource.TenantID == nil || *targetResource.TenantID != tenantID {
		return fmt.Errorf("cannot delete resource from another tenant")
	}

	// Soft delete tenant resources
	return s.repo.SoftDelete(ctx, id)
}

func (s *ResourceServiceImpl) DiscoverResourcesByType(ctx context.Context, tenantID primitive.ObjectID, app models.App, resourceType ResourceType) ([]Resource, error) {
	filter := ResourceFilter{
		TenantID:      &tenantID,
		App:           &app,
		Type:          &resourceType,
		IncludeGlobal: true,
	}

	return s.repo.FindAll(ctx, filter)
}

func (s *ResourceServiceImpl) GetGlobalResourcesForApp(ctx context.Context, app models.App) ([]Resource, error) {
	globalScope := ResourceScopeGlobal
	appScope := ResourceScopeApp

	// Get global resources
	globalFilter := ResourceFilter{
		App:   &app,
		Scope: &globalScope,
	}
	globalResources, err := s.repo.FindAll(ctx, globalFilter)
	if err != nil {
		return nil, err
	}

	// Get app-level resources
	appFilter := ResourceFilter{
		App:   &app,
		Scope: &appScope,
	}
	appResources, err := s.repo.FindAll(ctx, appFilter)
	if err != nil {
		return nil, err
	}

	// Combine both
	return append(globalResources, appResources...), nil
}
