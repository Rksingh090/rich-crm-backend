package app

import (
	"github.com/gofiber/fiber/v2"
)

type AppController struct {
	service AppService
}

func NewAppController(service AppService) *AppController {
	return &AppController{service: service}
}

func (c *AppController) ListApps(ctx *fiber.Ctx) error {
	apps, err := c.service.ListApps(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(apps)
}
