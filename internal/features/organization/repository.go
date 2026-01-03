package organization

import (
	"context"
	"go-crm/internal/common/models"
	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type OrganizationRepository interface {
	Create(ctx context.Context, org *models.Organization) error
	FindByID(ctx context.Context, id string) (*models.Organization, error)
	FindByName(ctx context.Context, name string) (*models.Organization, error)
	Update(ctx context.Context, org *models.Organization) error
}

type OrganizationRepositoryImpl struct {
	Collection *mongo.Collection
}

func NewOrganizationRepository(mongodb *database.MongodbDB) OrganizationRepository {
	return &OrganizationRepositoryImpl{
		Collection: mongodb.DB.Collection("organizations"),
	}
}

func (r *OrganizationRepositoryImpl) Create(ctx context.Context, org *models.Organization) error {
	_, err := r.Collection.InsertOne(ctx, org)
	return err
}

func (r *OrganizationRepositoryImpl) FindByID(ctx context.Context, id string) (*models.Organization, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var org models.Organization
	err = r.Collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&org)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *OrganizationRepositoryImpl) FindByName(ctx context.Context, name string) (*models.Organization, error) {
	var org models.Organization
	err := r.Collection.FindOne(ctx, bson.M{"name": name}).Decode(&org)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *OrganizationRepositoryImpl) Update(ctx context.Context, org *models.Organization) error {
	filter := bson.M{"_id": org.ID}
	update := bson.M{"$set": org}
	_, err := r.Collection.UpdateOne(ctx, filter, update)
	return err
}
