package api

import (
	"go-crm/internal/controllers"

	"github.com/gofiber/fiber/v2"
)

type SettingsApi struct {
	Controller *controllers.SettingsController
}

func NewSettingsApi(controller *controllers.SettingsController) *SettingsApi {
	return &SettingsApi{
		Controller: controller,
	}
}

func (a *SettingsApi) Setup(app *fiber.App) {
	// TODO: Add Auth Middleware once config is available or passed in
	group := app.Group("/api/settings")
	group.Get("/email", a.Controller.GetEmailConfig)
	group.Put("/email", a.Controller.UpdateEmailConfig)
}
