package app

import (
	"context"
	"go-crm/internal/common/models"
	"go-crm/internal/database"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AppRepository interface {
	Create(ctx context.Context, app *models.Application) error
	GetByName(ctx context.Context, name string) (*models.Application, error)
	List(ctx context.Context) ([]models.Application, error)
	Update(ctx context.Context, app *models.Application) error
}

type appRepositoryImpl struct {
	mongodb *database.MongodbDB
}

func NewAppRepository(mongodb *database.MongodbDB) AppRepository {
	return &appRepositoryImpl{
		mongodb: mongodb,
	}
}

func (r *appRepositoryImpl) collection() *mongo.Collection {
	return r.mongodb.GetControlPlaneDB().Collection("apps")
}

func (r *appRepositoryImpl) Create(ctx context.Context, app *models.Application) error {
	if app.CreatedAt.IsZero() {
		app.CreatedAt = time.Now()
	}
	app.UpdatedAt = time.Now()
	_, err := r.collection().InsertOne(ctx, app)
	return err
}

func (r *appRepositoryImpl) GetByName(ctx context.Context, name string) (*models.Application, error) {
	var app models.Application
	err := r.collection().FindOne(ctx, bson.M{"name": name}).Decode(&app)
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *appRepositoryImpl) List(ctx context.Context) ([]models.Application, error) {
	cursor, err := r.collection().Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var apps []models.Application
	if err = cursor.All(ctx, &apps); err != nil {
		return nil, err
	}
	return apps, nil
}

func (r *appRepositoryImpl) Update(ctx context.Context, app *models.Application) error {
	app.UpdatedAt = time.Now()
	_, err := r.collection().UpdateOne(
		ctx,
		bson.M{"_id": app.ID},
		bson.M{"$set": app},
		options.Update().SetUpsert(true),
	)
	return err
}
