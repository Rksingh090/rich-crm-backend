package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type UserApi struct {
	controller *controllers.UserController
	config     *config.Config
}

func NewUserApi(controller *controllers.UserController, config *config.Config) *UserApi {
	return &UserApi{
		controller: controller,
		config:     config,
	}
}

// Setup registers all user-related routes
func (h *UserApi) Setup(app *fiber.App) {
	// User routes group with auth and admin middleware
	users := app.Group("/users", middleware.AuthMiddleware(h.config.SkipAuth), middleware.AdminMiddleware())

	// User CRUD
	users.Get("/", h.controller.ListUsers)
	users.Get("/:id", h.controller.GetUser)
	users.Put("/:id", h.controller.UpdateUser)
	users.Delete("/:id", h.controller.DeleteUser)

	// User sub-resources
	users.Put("/:id/roles", h.controller.UpdateUserRoles)
	users.Put("/:id/status", h.controller.UpdateUserStatus)
}
