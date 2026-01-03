package admin

import (
	"go-crm/internal/config"
	"go-crm/internal/features/role"

	"github.com/gofiber/fiber/v2"
)

type AdminApi struct {
	App         *fiber.App
	Controller  *AdminController
	config      *config.Config
	roleService role.RoleService
}

func NewAdminApi(roleService role.RoleService, config *config.Config, controller *AdminController) *AdminApi {
	return &AdminApi{
		roleService: roleService,
		config:      config,
		Controller:  controller,
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
