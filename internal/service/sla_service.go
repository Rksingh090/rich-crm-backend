package service

import (
	"context"
	"errors"

	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SLAPervice defines the interface for SLA policy management
type SLAPervice interface { // Typo in original interface name? No SLAService
	CreatePolicy(ctx context.Context, policy *models.SLAPolicy) error
	GetPolicy(ctx context.Context, id string) (*models.SLAPolicy, error)
	ListPolicies(ctx context.Context) ([]models.SLAPolicy, error)
	UpdatePolicy(ctx context.Context, id string, updates map[string]interface{}) error
	DeletePolicy(ctx context.Context, id string) error
}

// Wait, original file had `SLAService`. I should stick to that naming.
type SLAService interface {
	CreatePolicy(ctx context.Context, policy *models.SLAPolicy) error
	GetPolicy(ctx context.Context, id string) (*models.SLAPolicy, error)
	ListPolicies(ctx context.Context) ([]models.SLAPolicy, error)
	UpdatePolicy(ctx context.Context, id string, updates map[string]interface{}) error
	DeletePolicy(ctx context.Context, id string) error
}

// SLAServiceImpl implements SLAService
type SLAServiceImpl struct {
	SLAPolicyRepo repository.SLAPolicyRepository
}

// NewSLAService creates a new SLA service
func NewSLAService(slaPolicyRepo repository.SLAPolicyRepository) SLAService {
	return &SLAServiceImpl{
		SLAPolicyRepo: slaPolicyRepo,
	}
}

// CreatePolicy creates a new SLA policy
func (s *SLAServiceImpl) CreatePolicy(ctx context.Context, policy *models.SLAPolicy) error {
	return s.SLAPolicyRepo.Create(ctx, policy)
}

// GetPolicy retrieves an SLA policy by ID
func (s *SLAServiceImpl) GetPolicy(ctx context.Context, id string) (*models.SLAPolicy, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid policy ID")
	}

	return s.SLAPolicyRepo.FindByID(ctx, objID)
}

// ListPolicies retrieves all SLA policies
func (s *SLAServiceImpl) ListPolicies(ctx context.Context) ([]models.SLAPolicy, error) {
	return s.SLAPolicyRepo.FindAll(ctx)
}

// UpdatePolicy updates an SLA policy
func (s *SLAServiceImpl) UpdatePolicy(ctx context.Context, id string, updates map[string]interface{}) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid policy ID")
	}

	bsonUpdates := bson.M{}
	for k, v := range updates {
		bsonUpdates[k] = v
	}

	return s.SLAPolicyRepo.Update(ctx, objID, bsonUpdates)
}

// DeletePolicy deletes an SLA policy
func (s *SLAServiceImpl) DeletePolicy(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid policy ID")
	}

	return s.SLAPolicyRepo.Delete(ctx, objID)
}
