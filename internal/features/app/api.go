package app

import (
	"github.com/gofiber/fiber/v2"
)

type AppApi struct {
	controller *AppController
}

func NewAppApi(controller *AppController) *AppApi {
	return &AppApi{controller: controller}
}

func (h *AppApi) Setup(app *fiber.App) {
	app.Get("/api/apps", h.controller.ListApps)
}
