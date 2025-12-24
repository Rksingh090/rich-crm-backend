package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type RoleApi struct {
	controller  *controllers.RoleController
	config      *config.Config
	roleService service.RoleService
}

func NewRoleApi(controller *controllers.RoleController, cfg *config.Config, roleService service.RoleService) *RoleApi {
	return &RoleApi{
		controller:  controller,
		config:      cfg,
		roleService: roleService,
	}
}

// Setup registers role routes
func (h *RoleApi) Setup(app *fiber.App) {
	// Role routes group with auth middleware
	roles := app.Group("/roles", middleware.AuthMiddleware(h.config.SkipAuth))

	// Role CRUD - require "roles" module permissions
	roles.Get("/", middleware.RequirePermission(h.roleService, "roles", "read"), h.controller.ListRoles)
	roles.Post("/", middleware.RequirePermission(h.roleService, "roles", "create"), h.controller.CreateRole)
	roles.Get("/:id", middleware.RequirePermission(h.roleService, "roles", "read"), h.controller.GetRole)
	roles.Put("/:id", middleware.RequirePermission(h.roleService, "roles", "update"), h.controller.UpdateRole)
	roles.Delete("/:id", middleware.RequirePermission(h.roleService, "roles", "delete"), h.controller.DeleteRole)
}
