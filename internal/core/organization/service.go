package organization

import (
	"context"
	"go-crm/internal/common/models"
)

type OrganizationService interface {
	CreateOrganization(ctx context.Context, org *models.Organization) error
	GetOrganization(ctx context.Context, id string) (*models.Organization, error)
	ListOrganizations(ctx context.Context, filter map[string]interface{}) ([]models.Organization, error)
	UpdateOrganization(ctx context.Context, org *models.Organization) error
	DeleteOrganization(ctx context.Context, id string) error
}

type OrganizationServiceImpl struct {
	repo OrganizationRepository
}

func NewOrganizationService(repo OrganizationRepository) OrganizationService {
	return &OrganizationServiceImpl{
		repo: repo,
	}
}

func (s *OrganizationServiceImpl) CreateOrganization(ctx context.Context, org *models.Organization) error {
	return s.repo.Create(ctx, org)
}

func (s *OrganizationServiceImpl) GetOrganization(ctx context.Context, id string) (*models.Organization, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *OrganizationServiceImpl) ListOrganizations(ctx context.Context, filter map[string]interface{}) ([]models.Organization, error) {
	return s.repo.List(ctx, filter)
}

func (s *OrganizationServiceImpl) UpdateOrganization(ctx context.Context, org *models.Organization) error {
	return s.repo.Update(ctx, org)
}

func (s *OrganizationServiceImpl) DeleteOrganization(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
