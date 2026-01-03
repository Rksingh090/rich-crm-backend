package automation

import (
	"go-crm/internal/common/api"
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type AutomationApi struct {
	controller *AutomationController
	config     *config.Config
}

func NewAutomationApi(controller *AutomationController, config *config.Config) api.Route {
	return &AutomationApi{
		controller: controller,
		config:     config,
	}
}

func (h *AutomationApi) Setup(app *fiber.App) {
	group := app.Group("/api/automation", middleware.AuthMiddleware(h.config.SkipAuth))

	group.Get("/rules", h.controller.ListRules)
	group.Get("/rules/:id", h.controller.GetRule)
	group.Post("/rules", h.controller.CreateRule)
	group.Put("/rules/:id", h.controller.UpdateRule)
	group.Delete("/rules/:id", h.controller.DeleteRule)
}
