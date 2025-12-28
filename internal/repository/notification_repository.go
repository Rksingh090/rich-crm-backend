package repository

import (
	"context"
	"go-crm/internal/database"
	"go-crm/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *models.Notification) error
	GetByUserID(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]models.Notification, int64, error)
	GetUnreadCount(ctx context.Context, userID primitive.ObjectID) (int64, error)
	MarkAsRead(ctx context.Context, id primitive.ObjectID, userID primitive.ObjectID) error
	MarkAllAsRead(ctx context.Context, userID primitive.ObjectID) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

type NotificationRepositoryImpl struct {
	collection *mongo.Collection
}

func NewNotificationRepository(db *database.MongodbDB) NotificationRepository {
	return &NotificationRepositoryImpl{
		collection: db.DB.Collection("notifications"),
	}
}

func (r *NotificationRepositoryImpl) Create(ctx context.Context, notification *models.Notification) error {
	notification.CreatedAt = time.Now()
	notification.IsRead = false
	result, err := r.collection.InsertOne(ctx, notification)
	if err != nil {
		return err
	}
	notification.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *NotificationRepositoryImpl) GetByUserID(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]models.Notification, int64, error) {
	skip := (page - 1) * limit
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(limit)

	filter := bson.M{"user_id": userID}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

func (r *NotificationRepositoryImpl) GetUnreadCount(ctx context.Context, userID primitive.ObjectID) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{
		"user_id": userID,
		"is_read": false,
	})
}

func (r *NotificationRepositoryImpl) MarkAsRead(ctx context.Context, id primitive.ObjectID, userID primitive.ObjectID) error {
	now := time.Now()
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": id, "user_id": userID},
		bson.M{
			"$set": bson.M{
				"is_read": true,
				"read_at": now,
			},
		},
	)
	return err
}

func (r *NotificationRepositoryImpl) MarkAllAsRead(ctx context.Context, userID primitive.ObjectID) error {
	now := time.Now()
	_, err := r.collection.UpdateMany(ctx,
		bson.M{"user_id": userID, "is_read": false},
		bson.M{
			"$set": bson.M{
				"is_read": true,
				"read_at": now,
			},
		},
	)
	return err
}

func (r *NotificationRepositoryImpl) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
