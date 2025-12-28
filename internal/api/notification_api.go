package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type NotificationApi struct {
	controller *controllers.NotificationController
	config     *config.Config
}

func NewNotificationApi(controller *controllers.NotificationController, config *config.Config) *NotificationApi {
	return &NotificationApi{
		controller: controller,
		config:     config,
	}
}

func (h *NotificationApi) Setup(app *fiber.App) {
	group := app.Group("/api/notifications", middleware.AuthMiddleware(h.config.SkipAuth))

	group.Get("/", h.controller.List)
	group.Get("/unread-count", h.controller.GetUnreadCount)
	group.Put("/:id/read", h.controller.MarkAsRead)
	group.Post("/mark-all-read", h.controller.MarkAllAsRead)
}
