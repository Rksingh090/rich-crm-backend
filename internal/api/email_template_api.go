package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type EmailTemplateApi struct {
	controller *controllers.EmailTemplateController
	config     *config.Config
}

func NewEmailTemplateApi(
	controller *controllers.EmailTemplateController,
	config *config.Config,
) *EmailTemplateApi {
	return &EmailTemplateApi{
		controller: controller,
		config:     config,
	}
}

// Setup registers all email template routes
func (h *EmailTemplateApi) Setup(app *fiber.App) {
	// Protected routes
	templates := app.Group("/api/email-templates", middleware.AuthMiddleware(h.config.SkipAuth))

	templates.Post("/", h.controller.Create)
	templates.Get("/", h.controller.List)
	templates.Get("/:id", h.controller.Get)
	templates.Put("/:id", h.controller.Update)
	templates.Delete("/:id", h.controller.Delete)

	// Helper to get fields for editor
	templates.Get("/fields/:module", h.controller.GetModuleFields)
}
