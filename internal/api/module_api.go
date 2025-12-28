package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type ModuleApi struct {
	moduleController *controllers.ModuleController
	recordController *controllers.RecordController
	config           *config.Config
}

func NewModuleApi(
	moduleController *controllers.ModuleController,
	recordController *controllers.RecordController,
	config *config.Config,
) *ModuleApi {
	return &ModuleApi{
		moduleController: moduleController,
		recordController: recordController,
		config:           config,
	}
}

// Setup registers all module and record-related routes
func (h *ModuleApi) Setup(app *fiber.App) {
	// Module routes group with auth middleware
	modules := app.Group("/api/modules", middleware.AuthMiddleware(h.config.SkipAuth))

	// Module CRUD
	modules.Post("/", h.moduleController.CreateModule)
	modules.Get("/", h.moduleController.ListModules)
	modules.Get("/:name", h.moduleController.GetModule)
	modules.Put("/:name", h.moduleController.UpdateModule)
	modules.Delete("/:name", h.moduleController.DeleteModule)

	// Record routes for modules
	modules.Get("/:name/records", h.recordController.ListRecords)
	modules.Post("/:name/records", h.recordController.CreateRecord)
	modules.Get("/:name/records/:id", h.recordController.GetRecord)
	modules.Put("/:name/records/:id", h.recordController.UpdateRecord)
	modules.Delete("/:name/records/:id", h.recordController.DeleteRecord)
}
