package email_template

import (
	"context"
	"time"

	"go-crm/internal/common/models"
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

	filter := bson.M{"_id": oid}

	// Add tenant check
	if tenantID, ok := ctx.Value(models.TenantIDKey).(primitive.ObjectID); ok {
		filter["tenant_id"] = tenantID
	}

	var template EmailTemplate
	err = r.collection.FindOne(ctx, filter).Decode(&template)
	if err != nil {
		return nil, err
	}

	return &template, nil
}

func (r *EmailTemplateRepositoryImpl) List(ctx context.Context, moduleName string) ([]EmailTemplate, error) {
	filter := bson.M{}

	// Add tenant check
	if tenantID, ok := ctx.Value(models.TenantIDKey).(primitive.ObjectID); ok {
		filter["tenant_id"] = tenantID
	}

	if moduleName != "" {
		if _, ok := filter["tenant_id"]; ok {
			// If we have tenant_id, we can just add module_name to the filter
			filter["module_name"] = moduleName
		} else {
			// Original logic for when tenant_id might not be present (though it should be)
			// Retaining logic but combining with tenant check if present
			orFilter := []bson.M{
				{"module_name": moduleName},
				{"module_name": ""},
				{"module_name": bson.M{"$exists": false}},
			}
			filter["$or"] = orFilter
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

	// Add tenant check
	if tenantID, ok := ctx.Value(models.TenantIDKey).(primitive.ObjectID); ok {
		filter["tenant_id"] = tenantID
	}

	update := bson.M{"$set": template}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *EmailTemplateRepositoryImpl) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": oid}

	// Add tenant check
	if tenantID, ok := ctx.Value(models.TenantIDKey).(primitive.ObjectID); ok {
		filter["tenant_id"] = tenantID
	}

	_, err = r.collection.DeleteOne(ctx, filter)
	return err
}
