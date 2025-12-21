package repository

import (
	"context"

	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoModuleRepository struct {
	Collection *mongo.Collection
	DB         *mongo.Database
}

func NewMongoModuleRepository(db *mongo.Database) *MongoModuleRepository {
	return &MongoModuleRepository{
		Collection: db.Collection("modules"),
		DB:         db,
	}
}

func (r *MongoModuleRepository) Create(ctx context.Context, module *models.Module) error {
	_, err := r.Collection.InsertOne(ctx, module)
	return err
}

func (r *MongoModuleRepository) FindByName(ctx context.Context, name string) (*models.Module, error) {
	var module models.Module
	err := r.Collection.FindOne(ctx, bson.M{"name": name}).Decode(&module)
	if err != nil {
		return nil, err
	}
	return &module, nil
}

func (r *MongoModuleRepository) List(ctx context.Context) ([]models.Module, error) {
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

func (r *MongoModuleRepository) Update(ctx context.Context, module *models.Module) error {
	filter := bson.M{"name": module.Name}
	update := bson.M{"$set": module}
	_, err := r.Collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoModuleRepository) Delete(ctx context.Context, name string) error {
	_, err := r.Collection.DeleteOne(ctx, bson.M{"name": name})
	return err
}

func (r *MongoModuleRepository) DropCollection(ctx context.Context, name string) error {
	return r.DB.Collection(name).Drop(ctx)
}

func (r *MongoModuleRepository) FindUsingLookup(ctx context.Context, targetModule string) ([]models.Module, error) {
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
