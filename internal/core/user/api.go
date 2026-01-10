package user

import (
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type UserApi struct {
	controller  *UserController
	config      *config.Config
	roleService middleware.RoleService
}

func NewUserApi(controller *UserController, config *config.Config, roleService middleware.RoleService) *UserApi {
	return &UserApi{
		controller:  controller,
		config:      config,
		roleService: roleService,
	}
}

// Setup registers all user-related routes
func (h *UserApi) Setup(app *fiber.App) {
	// User routes group with auth middleware
	users := app.Group("/api/users", middleware.AuthMiddleware(h.config.SkipAuth))

	// User CRUD
	// Create: requires explicit create permission
	users.Post("/", middleware.RequirePermission(h.roleService, "crm.settings_users", "create"), h.controller.CreateUser)

	// List: requires explicit read permission (for listing everyone)
	users.Get("/", h.controller.ListUsers)

	// Get One: Dynamic permission check in controller (Self or Admin)
	users.Get("/:id", h.controller.GetUser)

	// Update/Delete: requires explicit permission
	users.Put("/:id", middleware.RequirePermission(h.roleService, "crm.settings_users", "update"), h.controller.UpdateUser)
	users.Delete("/:id", middleware.RequirePermission(h.roleService, "crm.settings_users", "delete"), h.controller.DeleteUser)

	// User sub-resources
	users.Put("/:id/roles", middleware.RequirePermission(h.roleService, "crm.settings_users", "update"), h.controller.UpdateUserRoles)
	users.Put("/:id/status", middleware.RequirePermission(h.roleService, "crm.settings_users", "update"), h.controller.UpdateUserStatus)

	// Invite (Protected)
	users.Post("/invite", middleware.RequirePermission(h.roleService, "crm.settings_users", "create"), h.controller.InviteUser)

	// Public Routes (Accept Invite) - Register directly on app to avoid AuthMiddleware
	app.Post("/api/users/accept-invite", h.controller.AcceptInvite)
}
