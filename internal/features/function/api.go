package function

import (
	"go-crm/internal/common/api"
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type FunctionApi struct {
	controller *FunctionController
	config     *config.Config
}

func NewFunctionApi(
	controller *FunctionController,
	config *config.Config,
) api.Route {
	return &FunctionApi{
		controller: controller,
		config:     config,
	}
}

func (h *FunctionApi) Setup(app *fiber.App) {
	functions := app.Group("/api/functions", middleware.AuthMiddleware(h.config.SkipAuth))

	functions.Post("/", h.controller.Create)
	functions.Get("/", h.controller.List)
	functions.Get("/:id", h.controller.Get)
	functions.Put("/:id", h.controller.Update)
	functions.Delete("/:id", h.controller.Delete)
	functions.Post("/:id/test", h.controller.TestFunction)
}
