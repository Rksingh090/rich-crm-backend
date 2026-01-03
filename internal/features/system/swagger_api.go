package system

import (
	"go-crm/internal/common/api"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

type SwaggerApi struct{}

func NewSwaggerApi() api.Route {
	return &SwaggerApi{}
}

func (h *SwaggerApi) Setup(app *fiber.App) {
	app.Get("/swagger/*", swagger.HandlerDefault)
}
