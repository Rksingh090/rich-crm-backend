package emails

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Repository struct {
	col *mongo.Collection
}

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{
		col: db.Collection("emails"),
	}
}

func (r *Repository) Create(ctx context.Context, email *Email) error {
	email.CreatedAt = time.Now()
	_, err := r.col.InsertOne(ctx, email)
	return err
}

func (r *Repository) UpdateStatus(
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
