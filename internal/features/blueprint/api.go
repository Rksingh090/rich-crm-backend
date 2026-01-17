package blueprint

import (
	"go-crm/internal/config"
	"go-crm/internal/core/role"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type BlueprintApi struct {
	controller  *Controller
	roleService role.RoleService
	config      *config.Config
}

func NewBlueprintApi(controller *Controller, roleService role.RoleService, config *config.Config) *BlueprintApi {
	return &BlueprintApi{
		controller:  controller,
		roleService: roleService,
		config:      config,
	}
}

func (h *BlueprintApi) Setup(app *fiber.App) {
	api := app.Group("/api/blueprints", middleware.AuthMiddleware(h.config.SkipAuth))

	// TODO: Add refined permissions. For now assuming admin/manager access or generic check.
	// We can add "blueprints" resource to permission system properly later.
	// For now using simple auth for exploration.

	api.Post("/", h.controller.Create)
	api.Put("/:id", h.controller.Update)
	api.Get("/:id", h.controller.Get)
	api.Get("/", h.controller.List)
	api.Delete("/:id", h.controller.Delete)

	// Execution Endpoint
	// This is a dedicated endpoint for testing or manual triggers.
	api.Post("/execute/:module/:id", h.controller.ExecuteTransition)
	api.Get("/transitions/:module/:id", h.controller.GetTransitions)
}
