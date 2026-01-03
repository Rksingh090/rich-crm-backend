package email

import (
	"context"
	"time"

	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type EmailRepository struct {
	col *mongo.Collection
}

func NewEmailRepository(db *database.MongodbDB) *EmailRepository {
	return &EmailRepository{
		col: db.DB.Collection("emails"),
	}
}

func (r *EmailRepository) Create(ctx context.Context, email *Email) error {
	email.CreatedAt = time.Now()
	_, err := r.col.InsertOne(ctx, email)
	return err
}

func (r *EmailRepository) UpdateStatus(
	ctx context.Context,
	id primitive.ObjectID,
	status EmailStatus,
	errorMsg string,
) error {
	update := bson.M{
		"$set": bson.M{
			"status":       status,
			"errorMessage": errorMsg,
		},
	}
	if status == EmailSent {
		update["$set"].(bson.M)["sentAt"] = time.Now()
	}
	_, err := r.col.UpdateByID(ctx, id, update)
	return err
}
