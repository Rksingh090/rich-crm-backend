package email_template

import (
	"context"
	"time"

	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type EmailTemplateRepository interface {
	Create(ctx context.Context, template *EmailTemplate) error
	GetByID(ctx context.Context, id string) (*EmailTemplate, error)
	List(ctx context.Context, moduleName string) ([]EmailTemplate, error)
	Update(ctx context.Context, template *EmailTemplate) error
	Delete(ctx context.Context, id string) error
}

type EmailTemplateRepositoryImpl struct {
	collection *mongo.Collection
}

func NewEmailTemplateRepository(db *database.MongodbDB) EmailTemplateRepository {
	return &EmailTemplateRepositoryImpl{
		collection: db.DB.Collection("email_templates"),
	}
}

func (r *EmailTemplateRepositoryImpl) Create(ctx context.Context, template *EmailTemplate) error {
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	if template.ID.IsZero() {
		template.ID = primitive.NewObjectID()
	}

	_, err := r.collection.InsertOne(ctx, template)
	return err
}

func (r *EmailTemplateRepositoryImpl) GetByID(ctx context.Context, id string) (*EmailTemplate, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var template EmailTemplate
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&template)
	if err != nil {
		return nil, err
	}

	return &template, nil
}

func (r *EmailTemplateRepositoryImpl) List(ctx context.Context, moduleName string) ([]EmailTemplate, error) {
	filter := bson.M{}

	if moduleName != "" {
		filter = bson.M{
			"$or": []bson.M{
				{"module_name": moduleName},
				{"module_name": ""},
				{"module_name": bson.M{"$exists": false}},
			},
		}
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var templates []EmailTemplate
	if err = cursor.All(ctx, &templates); err != nil {
		return nil, err
	}

	return templates, nil
}

func (r *EmailTemplateRepositoryImpl) Update(ctx context.Context, template *EmailTemplate) error {
	template.UpdatedAt = time.Now()

	filter := bson.M{"_id": template.ID}
	update := bson.M{"$set": template}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *EmailTemplateRepositoryImpl) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
