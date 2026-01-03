package email_template

import (
	"go-crm/internal/common/api"
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type EmailTemplateApi struct {
	controller *EmailTemplateController
	config     *config.Config
}

func NewEmailTemplateApi(
	controller *EmailTemplateController,
	config *config.Config,
) api.Route {
	return &EmailTemplateApi{
		controller: controller,
		config:     config,
	}
}

func (h *EmailTemplateApi) Setup(app *fiber.App) {
	templates := app.Group("/api/email-templates", middleware.AuthMiddleware(h.config.SkipAuth))

	templates.Post("/", h.controller.Create)
	templates.Get("/", h.controller.List)
	templates.Get("/:id", h.controller.Get)
	templates.Put("/:id", h.controller.Update)
	templates.Delete("/:id", h.controller.Delete)
	templates.Post("/:id/test", h.controller.SendTestEmail)

	templates.Get("/fields/:module", h.controller.GetModuleFields)
}
