package controllers

import (
	"go-crm/internal/service"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationController struct {
	service service.NotificationService
}

func NewNotificationController(service service.NotificationService) *NotificationController {
	return &NotificationController{
		service: service,
	}
}

// List retrieves notifications
// @Summary List notifications
// @Description Get a list of notifications for the current user
// @Tags Notifications
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/notifications [get]
func (c *NotificationController) List(ctx *fiber.Ctx) error {
	userIDStr := ctx.Locals("userID").(string)
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	page, _ := strconv.ParseInt(ctx.Query("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.Query("limit", "10"), 10, 64)

	notifications, total, err := c.service.GetUserNotifications(ctx.Context(), userID, page, limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{
		"data":  notifications,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetUnreadCount retrieves unread notification count
// @Summary Get unread count
// @Description Get the number of unread notifications
// @Tags Notifications
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/notifications/unread-count [get]
func (c *NotificationController) GetUnreadCount(ctx *fiber.Ctx) error {
	userIDStr := ctx.Locals("userID").(string)
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	count, err := c.service.GetUnreadCount(ctx.Context(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"count": count})
}

// MarkAsRead marks a notification as read
// @Summary Mark notification as read
// @Description Mark a specific notification as read
// @Tags Notifications
// @Accept json
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/notifications/{id}/read [put]
func (c *NotificationController) MarkAsRead(ctx *fiber.Ctx) error {
	userIDStr := ctx.Locals("userID").(string)
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	id := ctx.Params("id")
	if err := c.service.MarkAsRead(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"status": "success"})
}

// MarkAllAsRead marks all notifications as read
// @Summary Mark all as read
// @Description Mark all notifications for the current user as read
// @Tags Notifications
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/notifications/mark-all-read [post]
func (c *NotificationController) MarkAllAsRead(ctx *fiber.Ctx) error {
	userIDStr := ctx.Locals("userID").(string)
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	if err := c.service.MarkAllAsRead(ctx.Context(), userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"status": "success"})
}
