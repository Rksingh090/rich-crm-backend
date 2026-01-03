package notification

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationController struct {
	service NotificationService
}

func NewNotificationController(service NotificationService) *NotificationController {
	return &NotificationController{
		service: service,
	}
}

// List godoc
// List godoc
// @Summary List notifications
// @Description List user notifications with pagination
// @Tags notifications
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/notifications [get]
func (c *NotificationController) List(ctx *fiber.Ctx) error {
	userIDStr := ctx.Locals("user_id").(string)
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	page, _ := strconv.ParseInt(ctx.Query("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.Query("limit", "10"), 10, 64)

	notifications, total, err := c.service.GetUserNotifications(ctx.UserContext(), userID, page, limit)
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

// GetUnreadCount godoc
// GetUnreadCount godoc
// @Summary Get unread count
// @Description Get the count of unread notifications
// @Tags notifications
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/notifications/unread-count [get]
func (c *NotificationController) GetUnreadCount(ctx *fiber.Ctx) error {
	userIDStr := ctx.Locals("user_id").(string)
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	count, err := c.service.GetUnreadCount(ctx.UserContext(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"count": count})
}

// MarkAsRead godoc
// MarkAsRead godoc
// @Summary Mark notification as read
// @Description Mark a specific notification as read
// @Tags notifications
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/notifications/{id}/read [put]
func (c *NotificationController) MarkAsRead(ctx *fiber.Ctx) error {
	userIDStr := ctx.Locals("user_id").(string)
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	id := ctx.Params("id")
	if err := c.service.MarkAsRead(ctx.UserContext(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"status": "success"})
}

// MarkAllAsRead godoc
// MarkAllAsRead godoc
// @Summary Mark all as read
// @Description Mark all notifications as read for the user
// @Tags notifications
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/notifications/read-all [put]
func (c *NotificationController) MarkAllAsRead(ctx *fiber.Ctx) error {
	userIDStr := ctx.Locals("user_id").(string)
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	if err := c.service.MarkAllAsRead(ctx.UserContext(), userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"status": "success"})
}
