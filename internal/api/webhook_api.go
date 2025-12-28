package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type WebhookApi struct {
	controller  *controllers.WebhookController
	config      *config.Config
	roleService service.RoleService
}

func NewWebhookApi(controller *controllers.WebhookController, config *config.Config, roleService service.RoleService) *WebhookApi {
	return &WebhookApi{
		controller:  controller,
		config:      config,
		roleService: roleService,
	}
}

// Setup registers all webhook routes
func (h *WebhookApi) Setup(app *fiber.App) {
	webhooks := app.Group("/api/webhooks", middleware.AuthMiddleware(h.config.SkipAuth))

	webhooks.Post("/", middleware.RequirePermission(h.roleService, "webhooks", "create"), h.controller.CreateWebhook)
	webhooks.Get("/", middleware.RequirePermission(h.roleService, "webhooks", "read"), h.controller.ListWebhooks)
	webhooks.Get("/:id", middleware.RequirePermission(h.roleService, "webhooks", "read"), h.controller.GetWebhook)
	webhooks.Put("/:id", middleware.RequirePermission(h.roleService, "webhooks", "update"), h.controller.UpdateWebhook)
	webhooks.Delete("/:id", middleware.RequirePermission(h.roleService, "webhooks", "delete"), h.controller.DeleteWebhook)
}
