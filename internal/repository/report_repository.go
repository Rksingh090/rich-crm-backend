package repository

import (
	"context"
	"go-crm/internal/database"
	"go-crm/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ReportRepository interface {
	Create(ctx context.Context, report *models.Report) error
	Get(ctx context.Context, id string) (*models.Report, error)
	List(ctx context.Context) ([]models.Report, error)
	Update(ctx context.Context, id string, report *models.Report) error
	Delete(ctx context.Context, id string) error
}

type ReportRepositoryImpl struct {
	Collection *mongo.Collection
}

func NewReportRepository(db *database.MongodbDB) ReportRepository {
	return &ReportRepositoryImpl{
		Collection: db.DB.Collection("reports"),
	}
}

func (r *ReportRepositoryImpl) Create(ctx context.Context, report *models.Report) error {
	report.CreatedAt = time.Now()
	report.UpdatedAt = time.Now()
	_, err := r.Collection.InsertOne(ctx, report)
	return err
}

func (r *ReportRepositoryImpl) Get(ctx context.Context, id string) (*models.Report, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var report models.Report
	err = r.Collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&report)
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *ReportRepositoryImpl) List(ctx context.Context) ([]models.Report, error) {
	cursor, err := r.Collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reports []models.Report
	if err := cursor.All(ctx, &reports); err != nil {
		return nil, err
	}
	return reports, nil
}

func (r *ReportRepositoryImpl) Update(ctx context.Context, id string, report *models.Report) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	report.UpdatedAt = time.Now()
	// Exclude ID from update
	update := bson.M{
		"$set": bson.M{
			"name":        report.Name,
			"description": report.Description,
			"module_id":   report.ModuleID,
			"columns":     report.Columns,
			"filters":     report.Filters,
			"updated_at":  report.UpdatedAt,
			"updated_by":  report.UpdatedBy,
		},
	}
	_, err = r.Collection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	return err
}

func (r *ReportRepositoryImpl) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.Collection.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
