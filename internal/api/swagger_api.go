package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

type SwaggerApi struct{}

func NewSwaggerApi() *SwaggerApi {
	return &SwaggerApi{}
}

// Setup registers Swagger UI route
func (h *SwaggerApi) Setup(app *fiber.App) {
	app.Get("/swagger/*", swagger.HandlerDefault)
}
