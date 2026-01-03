package module

import (
	"go-crm/internal/config"
	"go-crm/internal/features/role"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type ModuleApi struct {
	moduleController *ModuleController
	config           *config.Config
	roleService      role.RoleService
}

func NewModuleApi(
	moduleController *ModuleController,
	config *config.Config,
	roleService role.RoleService,
) *ModuleApi {
	return &ModuleApi{
		moduleController: moduleController,
		config:           config,
		roleService:      roleService,
	}
}

// Setup registers all module-related routes
func (h *ModuleApi) Setup(app *fiber.App) {
	// Module routes group with auth middleware
	modules := app.Group("/api/modules", middleware.AuthMiddleware(h.config.SkipAuth))

	// Module CRUD (Schema Management) - Protected by "modules" permission
	modules.Post("/", middleware.RequirePermission(h.roleService, "modules", "create"), h.moduleController.CreateModule)
	// ListModules handles its own permission checking (granular)
	modules.Get("/", h.moduleController.ListModules)
	modules.Get("/:name", h.moduleController.GetModule)
	modules.Put("/:name", h.moduleController.UpdateModule)
	modules.Delete("/:name", h.moduleController.DeleteModule)
}
