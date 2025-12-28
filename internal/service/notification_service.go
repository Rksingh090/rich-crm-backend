package service

import (
	"context"
	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationService interface {
	CreateNotification(ctx context.Context, userID primitive.ObjectID, title, message string, notifType models.NotificationType, link string) error
	GetUserNotifications(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]models.Notification, int64, error)
	GetUnreadCount(ctx context.Context, userID primitive.ObjectID) (int64, error)
	MarkAsRead(ctx context.Context, id string, userID primitive.ObjectID) error
	MarkAllAsRead(ctx context.Context, userID primitive.ObjectID) error
}

type NotificationServiceImpl struct {
	repo repository.NotificationRepository
}

func NewNotificationService(repo repository.NotificationRepository) NotificationService {
	return &NotificationServiceImpl{
		repo: repo,
	}
}

func (s *NotificationServiceImpl) CreateNotification(ctx context.Context, userID primitive.ObjectID, title, message string, notifType models.NotificationType, link string) error {
	notification := &models.Notification{
		UserID:  userID,
		Title:   title,
		Message: message,
		Type:    notifType,
		Link:    link,
	}
	return s.repo.Create(ctx, notification)
}

func (s *NotificationServiceImpl) GetUserNotifications(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]models.Notification, int64, error) {
	return s.repo.GetByUserID(ctx, userID, page, limit)
}

func (s *NotificationServiceImpl) GetUnreadCount(ctx context.Context, userID primitive.ObjectID) (int64, error) {
	return s.repo.GetUnreadCount(ctx, userID)
}

func (s *NotificationServiceImpl) MarkAsRead(ctx context.Context, id string, userID primitive.ObjectID) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	return s.repo.MarkAsRead(ctx, objID, userID)
}

func (s *NotificationServiceImpl) MarkAllAsRead(ctx context.Context, userID primitive.ObjectID) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}
