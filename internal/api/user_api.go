package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type UserApi struct {
	controller  *controllers.UserController
	config      *config.Config
	roleService service.RoleService
}

func NewUserApi(controller *controllers.UserController, config *config.Config, roleService service.RoleService) *UserApi {
	return &UserApi{
		controller:  controller,
		config:      config,
		roleService: roleService,
	}
}

// Setup registers all user-related routes
func (h *UserApi) Setup(app *fiber.App) {
	// User routes group with auth middleware
	users := app.Group("/users", middleware.AuthMiddleware(h.config.SkipAuth))

	// User CRUD - require "users" module permissions
	users.Get("/", middleware.RequirePermission(h.roleService, "users", "read"), h.controller.ListUsers)
	users.Get("/:id", middleware.RequirePermission(h.roleService, "users", "read"), h.controller.GetUser)
	users.Put("/:id", middleware.RequirePermission(h.roleService, "users", "update"), h.controller.UpdateUser)
	users.Delete("/:id", middleware.RequirePermission(h.roleService, "users", "delete"), h.controller.DeleteUser)

	// User sub-resources
	users.Put("/:id/roles", middleware.RequirePermission(h.roleService, "users", "update"), h.controller.UpdateUserRoles)
	users.Put("/:id/status", middleware.RequirePermission(h.roleService, "users", "update"), h.controller.UpdateUserStatus)
}
