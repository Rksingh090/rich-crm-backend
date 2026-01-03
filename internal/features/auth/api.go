package auth

import (
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type AuthApi struct {
	controller *AuthController
	config     *config.Config
}

func NewAuthApi(controller *AuthController, config *config.Config) *AuthApi {
	return &AuthApi{
		controller: controller,
		config:     config,
	}
}

// Setup registers all auth-related routes
func (h *AuthApi) Setup(app *fiber.App) {
	// Public routes
	app.Post("/api/register", h.controller.Register)
	app.Post("/api/login", h.controller.Login)

	// Protected route example
	app.Get("/api/protected", middleware.AuthMiddleware(h.config.SkipAuth), h.protectedRoute)
}

func (h *AuthApi) protectedRoute(c *fiber.Ctx) error {
	return c.SendString("You are authenticated!")
}
