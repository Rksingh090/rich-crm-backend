package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type RoleApi struct {
	controller *controllers.RoleController
	config     *config.Config
}

func NewRoleApi(controller *controllers.RoleController, cfg *config.Config) *RoleApi {
	return &RoleApi{
		controller: controller,
		config:     cfg,
	}
}

// Setup registers role routes
func (h *RoleApi) Setup(app *fiber.App) {
	// Role routes group with auth and admin middleware
	roles := app.Group("/roles", middleware.AuthMiddleware(h.config.SkipAuth), middleware.AdminMiddleware())

	roles.Get("/", h.controller.ListRoles)
	roles.Post("/", h.controller.CreateRole)
	roles.Get("/:id", h.controller.GetRole)
	roles.Put("/:id", h.controller.UpdateRole)
	roles.Delete("/:id", h.controller.DeleteRole)
}
