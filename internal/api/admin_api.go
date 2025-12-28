package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type AdminApi struct {
	App         *fiber.App
	Controller  *controllers.AdminController
	config      *config.Config
	roleService service.RoleService
}

func NewAdminApi(roleService service.RoleService, config *config.Config) *AdminApi {
	return &AdminApi{
		roleService: roleService,
		config:      config,
	}
}

// Setup registers admin-related routes
func (h *AdminApi) Setup(app *fiber.App) {
	// Admin route with RBAC
	app.Get("/api/admin",
		h.Controller.WelcomeAdmin,
	)
	app.Post("/api/admin/handle-webhook",
		h.Controller.HandleWebhook,
	)
}
