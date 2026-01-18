package organization

import (
	"context"
	"go-crm/internal/common/models"
	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OrganizationRepository interface {
	Create(ctx context.Context, org *models.Organization) error
	FindByID(ctx context.Context, id string) (*models.Organization, error)
	FindByName(ctx context.Context, name string) (*models.Organization, error)
	List(ctx context.Context, filter map[string]any) ([]models.Organization, error)
	Update(ctx context.Context, org *models.Organization) error
	Delete(ctx context.Context, id string) error
	EnsureIndexes(ctx context.Context) error
}

type OrganizationRepositoryImpl struct {
	DB *database.MongodbDB
}

func NewOrganizationRepository(mongodb *database.MongodbDB) OrganizationRepository {
	return &OrganizationRepositoryImpl{
		DB: mongodb,
	}
}

func (r *OrganizationRepositoryImpl) getCollection() *mongo.Collection {
	return r.DB.GetControlPlaneDB().Collection("organizations")
}

func (r *OrganizationRepositoryImpl) Create(ctx context.Context, org *models.Organization) error {
	_, err := r.getCollection().InsertOne(ctx, org)
	return err
}

func (r *OrganizationRepositoryImpl) FindByID(ctx context.Context, id string) (*models.Organization, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var org models.Organization
	err = r.getCollection().FindOne(ctx, bson.M{"_id": objectID}).Decode(&org)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *OrganizationRepositoryImpl) FindByName(ctx context.Context, name string) (*models.Organization, error) {
	var org models.Organization
	err := r.getCollection().FindOne(ctx, bson.M{"name": name}).Decode(&org)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *OrganizationRepositoryImpl) Update(ctx context.Context, org *models.Organization) error {
	filter := bson.M{"_id": org.ID}
	update := bson.M{"$set": org}
	_, err := r.getCollection().UpdateOne(ctx, filter, update)
	return err
}

func (r *OrganizationRepositoryImpl) List(ctx context.Context, filter map[string]any) ([]models.Organization, error) {
	cursor, err := r.getCollection().Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orgs []models.Organization
	if err := cursor.All(ctx, &orgs); err != nil {
		return nil, err
	}
	return orgs, nil
}

func (r *OrganizationRepositoryImpl) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.getCollection().DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *OrganizationRepositoryImpl) EnsureIndexes(ctx context.Context) error {
	coll := r.getCollection()
	_, err := coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "owner_id", Value: 1}},
		},
	})
	return err
}
