package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ModuleApi struct {
	moduleController *controllers.ModuleController
	recordController *controllers.RecordController
	config           *config.Config
	roleService      service.RoleService
}

func NewModuleApi(
	moduleController *controllers.ModuleController,
	recordController *controllers.RecordController,
	config *config.Config,
	roleService service.RoleService,
) *ModuleApi {
	return &ModuleApi{
		moduleController: moduleController,
		recordController: recordController,
		config:           config,
		roleService:      roleService,
	}
}

// Setup registers all module and record-related routes
func (h *ModuleApi) Setup(app *fiber.App) {
	// Module routes group with auth middleware
	modules := app.Group("/api/modules", middleware.AuthMiddleware(h.config.SkipAuth))

	// Module CRUD (Schema Management) - Protected by "modules" permission
	modules.Post("/", middleware.RequirePermission(h.roleService, "modules", "create"), h.moduleController.CreateModule)
	modules.Get("/", middleware.RequirePermission(h.roleService, "modules", "read"), h.moduleController.ListModules)
	modules.Get("/:name", middleware.RequirePermission(h.roleService, "modules", "read"), h.moduleController.GetModule)
	modules.Put("/:name", middleware.RequirePermission(h.roleService, "modules", "update"), h.moduleController.UpdateModule)
	modules.Delete("/:name", middleware.RequirePermission(h.roleService, "modules", "delete"), h.moduleController.DeleteModule)

	// Record routes for modules
	modules.Get("/:name/records", h.recordController.ListRecords)
	modules.Post("/:name/records", h.recordController.CreateRecord)
	modules.Get("/:name/records/:id", h.recordController.GetRecord)
	modules.Put("/:name/records/:id", h.recordController.UpdateRecord)
	modules.Delete("/:name/records/:id", h.recordController.DeleteRecord)
}
