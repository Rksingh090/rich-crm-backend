package system

import (
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type DebugApi struct {
	controller *DebugController
	config     *config.Config
}

func NewDebugApi(controller *DebugController, cfg *config.Config) *DebugApi {
	return &DebugApi{
		controller: controller,
		config:     cfg,
	}
}

// Setup registers debug routes
func (h *DebugApi) Setup(app *fiber.App) {
	debug := app.Group("/api/debug", middleware.AuthMiddleware(h.config.SkipAuth))
	debug.Get("/me", h.controller.GetCurrentUser)
}
