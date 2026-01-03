package webhook

import (
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type WebhookApi struct {
	controller  *WebhookController
	config      *config.Config
	roleService middleware.RoleService
}

func NewWebhookApi(controller *WebhookController, config *config.Config, roleService middleware.RoleService) *WebhookApi {
	return &WebhookApi{
		controller:  controller,
		config:      config,
		roleService: roleService,
	}
}

func (h *WebhookApi) Setup(app *fiber.App) {
	webhooks := app.Group("/api/webhooks", middleware.AuthMiddleware(h.config.SkipAuth))

	webhooks.Post("/", middleware.RequirePermission(h.roleService, "settings", "update"), h.controller.CreateWebhook)
	webhooks.Get("/", middleware.RequirePermission(h.roleService, "settings", "read"), h.controller.ListWebhooks)
	webhooks.Get("/:id", middleware.RequirePermission(h.roleService, "settings", "read"), h.controller.GetWebhook)
	webhooks.Put("/:id", middleware.RequirePermission(h.roleService, "settings", "update"), h.controller.UpdateWebhook)
	webhooks.Delete("/:id", middleware.RequirePermission(h.roleService, "settings", "update"), h.controller.DeleteWebhook)
}
