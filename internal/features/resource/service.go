package resource

import (
	"context"
	"go-crm/internal/features/role"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ResourceService interface {
	ListResources(ctx context.Context) ([]Resource, error)
	ListSidebarResources(ctx context.Context, product string, location string) ([]Resource, error)
	GetSidebar(ctx context.Context, userID string) ([]Resource, error)
	SyncResources(ctx context.Context, resources []Resource) error
}

type ResourceServiceImpl struct {
	repo        ResourceRepository
	roleService role.RoleService
}

func NewResourceService(repo ResourceRepository, roleService role.RoleService) ResourceService {
	return &ResourceServiceImpl{
		repo:        repo,
		roleService: roleService,
	}
}

func (s *ResourceServiceImpl) ListResources(ctx context.Context) ([]Resource, error) {
	return s.repo.FindAll(ctx)
}

func (s *ResourceServiceImpl) SyncResources(ctx context.Context, resources []Resource) error {
	for _, res := range resources {
		// Auto-generate ID from product.key if not provided
		if res.ID == "" {
			res.ID = res.Product + "." + res.Key
		}

		// Try to find existing resource (may not have tenant_id yet)
		existing, err := s.repo.FindByID(ctx, res.ID)
		if err == nil && existing != nil {
			// Update existing resource
			res.TenantID = existing.TenantID // Preserve existing tenant_id if set
			res.CreatedAt = existing.CreatedAt
			res.UpdatedAt = time.Now()
			if err := s.repo.Update(ctx, &res); err != nil {
				// If update fails due to tenant mismatch, try direct update
				return err
			}
		} else {
			// Create new resource
			res.CreatedAt = time.Now()
			res.UpdatedAt = time.Now()
			if err := s.repo.Create(ctx, &res); err != nil {
				// Ignore duplicate key errors (resource already exists)
				if !strings.Contains(err.Error(), "duplicate key") {
					return err
				}
			}
		}
	}
	return nil
}

func (s *ResourceServiceImpl) GetSidebar(ctx context.Context, userID string) ([]Resource, error) {
	// 1. Fetch all resources
	allResources, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	// 2. Filter by Sidebar = true and Permissions
	var sidebarResources []Resource

	// Convert string userID to ObjectID
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	for _, res := range allResources {
		if !res.UI.Sidebar {
			continue
		}

		// Check Permission
		// Assuming RoleService has a CheckPermission method that takes userID
		// We will need to implement/expose this.
		// For now, using a placeholder logic or assuming the method exists.
		allowed, err := s.roleService.CheckPermission(ctx, uID, res.ID, "read")
		if err != nil {
			continue
		}

		if allowed {
			sidebarResources = append(sidebarResources, res)
		}
	}

	return sidebarResources, nil
}

func (s *ResourceServiceImpl) ListSidebarResources(ctx context.Context, product string, location string) ([]Resource, error) {
	return s.repo.FindSidebarResources(ctx, product, location)
}
