package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type RecordApi struct {
	controller *controllers.RecordController
	config     *config.Config
}

func NewRecordApi(controller *controllers.RecordController, config *config.Config) *RecordApi {
	return &RecordApi{
		controller: controller,
		config:     config,
	}
}

// Setup registers all record-related routes
// Note: Record routes are actually registered in ModuleApi as they are nested under /modules/:name/records
func (h *RecordApi) Setup(app *fiber.App) {
	// Record routes are handled by ModuleApi
	// This is just a placeholder to satisfy the Route interface
	// If you want standalone record routes, add them here with auth middleware
	records := app.Group("/records", middleware.AuthMiddleware(h.config.SkipAuth))
	_ = records // Placeholder to avoid unused variable error
}
