package email_template

import (
	"context"
	"fmt"
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
	db *database.MongodbDB
}

func NewEmailTemplateRepository(db *database.MongodbDB) EmailTemplateRepository {
	return &EmailTemplateRepositoryImpl{
		db: db,
	}
}

func (r *EmailTemplateRepositoryImpl) getCollection(ctx context.Context) (*mongo.Collection, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		// Try primitive.ObjectID
		if oid, ok := ctx.Value(models.TenantIDKey).(primitive.ObjectID); ok {
			tenantID = oid.Hex()
		}
	}

	if tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}

	return r.db.GetTenantDB(tenantID).Collection("email_templates"), nil
}

func (r *EmailTemplateRepositoryImpl) Create(ctx context.Context, template *EmailTemplate) error {
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	if template.ID.IsZero() {
		template.ID = primitive.NewObjectID()
	}

	col, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	_, err = col.InsertOne(ctx, template)
	return err
}

func (r *EmailTemplateRepositoryImpl) GetByID(ctx context.Context, id string) (*EmailTemplate, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	col, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": oid}

	// Add app check if present in context, but allow templates with no app field
	if appID, ok := ctx.Value(models.AppIDKey).(string); ok && appID != "" {
		filter["$or"] = []bson.M{
			{"app": appID},
			{"app": ""},
			{"app": bson.M{"$exists": false}},
		}
	}

	var template EmailTemplate
	err = col.FindOne(ctx, filter).Decode(&template)
	if err != nil {
		return nil, err
	}

	return &template, nil
}

func (r *EmailTemplateRepositoryImpl) List(ctx context.Context, moduleName string) ([]EmailTemplate, error) {
	col, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	filter := bson.M{}

	if moduleName != "" {
		filter["$or"] = []bson.M{
			{"module_name": moduleName},
			{"module_name": ""},
			{"module_name": bson.M{"$exists": false}},
		}
	}

	// Add app check
	if appID, ok := ctx.Value(models.AppIDKey).(string); ok && appID != "" {
		appFilter := []bson.M{
			{"app": appID},
			{"app": ""},
			{"app": bson.M{"$exists": false}},
		}
		if existingOr, ok := filter["$or"]; ok {
			// If we already have an OR for module_name, we need to AND it with the app OR
			filter = bson.M{
				"$and": []bson.M{
					{"$or": existingOr},
					{"$or": appFilter},
				},
			}
		} else {
			filter["$or"] = appFilter
		}
	}

	cursor, err := col.Find(ctx, filter)
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

	col, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": template.ID}
	update := bson.M{"$set": template}

	_, err = col.UpdateOne(ctx, filter, update)
	return err
}

func (r *EmailTemplateRepositoryImpl) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	col, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	_, err = col.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
