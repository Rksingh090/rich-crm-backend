package api

import (
	"github.com/gofiber/fiber/v2"
)

type HealthApi struct{}

func NewHealthApi() *HealthApi {
	return &HealthApi{}
}

// Setup registers health check route
func (h *HealthApi) Setup(app *fiber.App) {
	app.Get("/health", h.HealthCheck)
}

// HealthCheck godoc
// @Summary      Health Check
// @Description  Check if the server is up
// @Tags         health
// @Produce      plain
// @Success      200  {string}  string  "OK"
// @Router       /health [get]
func (h *HealthApi) HealthCheck(c *fiber.Ctx) error {
	return c.SendString("OK")
}
