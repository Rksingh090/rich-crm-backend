package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type AuditApi struct {
	controller *controllers.AuditController
	config     *config.Config
}

func NewAuditApi(controller *controllers.AuditController, config *config.Config) *AuditApi {
	return &AuditApi{
		controller: controller,
		config:     config,
	}
}

// Setup registers all audit-related routes
func (h *AuditApi) Setup(app *fiber.App) {
	// Audit logs route (protected)
	app.Get("/audit-logs", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.ListLogs)
}
