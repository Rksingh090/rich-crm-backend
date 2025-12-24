package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type AuthApi struct {
	controller *controllers.AuthController
	config     *config.Config
}

func NewAuthApi(controller *controllers.AuthController, config *config.Config) *AuthApi {
	return &AuthApi{
		controller: controller,
		config:     config,
	}
}

// Setup registers all auth-related routes
func (h *AuthApi) Setup(app *fiber.App) {
	// Public routes
	app.Post("/register", h.controller.Register)
	app.Post("/login", h.controller.Login)

	// Protected route example
	app.Get("/protected", middleware.AuthMiddleware(h.config.SkipAuth), h.protectedRoute)
}

func (h *AuthApi) protectedRoute(c *fiber.Ctx) error {
	return c.SendString("You are authenticated!")
}
