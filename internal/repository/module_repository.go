package repository

import (
	"context"

	"go-crm/internal/database"
	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ModuleRepository interface {
	Create(ctx context.Context, module *models.Module) error
	FindByName(ctx context.Context, name string) (*models.Module, error)
	List(ctx context.Context) ([]models.Module, error)
	Update(ctx context.Context, module *models.Module) error
	Delete(ctx context.Context, name string) error
	DropCollection(ctx context.Context, name string) error
	FindUsingLookup(ctx context.Context, targetModule string) ([]models.Module, error)
}

type ModuleRepositoryImpl struct {
	Collection *mongo.Collection
	DB         *mongo.Database
}

func NewModuleRepository(mongodb *database.MongodbDB) ModuleRepository {
	return &ModuleRepositoryImpl{
		Collection: mongodb.DB.Collection("modules"),
		DB:         mongodb.DB,
	}
}

func (r *ModuleRepositoryImpl) Create(ctx context.Context, module *models.Module) error {
	_, err := r.Collection.InsertOne(ctx, module)
	return err
}

func (r *ModuleRepositoryImpl) FindByName(ctx context.Context, name string) (*models.Module, error) {
	var module models.Module
	err := r.Collection.FindOne(ctx, bson.M{"name": name}).Decode(&module)
	if err != nil {
		return nil, err
	}
	return &module, nil
}

func (r *ModuleRepositoryImpl) List(ctx context.Context) ([]models.Module, error) {
	cursor, err := r.Collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var modules []models.Module
	if err = cursor.All(ctx, &modules); err != nil {
		return nil, err
	}
	return modules, nil
}

func (r *ModuleRepositoryImpl) Update(ctx context.Context, module *models.Module) error {
	filter := bson.M{"name": module.Name}
	update := bson.M{"$set": module}
	_, err := r.Collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *ModuleRepositoryImpl) Delete(ctx context.Context, name string) error {
	_, err := r.Collection.DeleteOne(ctx, bson.M{"name": name})
	return err
}

func (r *ModuleRepositoryImpl) DropCollection(ctx context.Context, name string) error {
	return r.DB.Collection(name).Drop(ctx)
}

func (r *ModuleRepositoryImpl) FindUsingLookup(ctx context.Context, targetModule string) ([]models.Module, error) {
	// Find modules that have at least one field where field.lookup.module == targetModule
	filter := bson.M{
		"fields": bson.M{
			"$elemMatch": bson.M{
				"type":          "lookup",
				"lookup.module": targetModule,
			},
		},
	}

	cursor, err := r.Collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var modules []models.Module
	if err = cursor.All(ctx, &modules); err != nil {
		return nil, err
	}
	return modules, nil
}
