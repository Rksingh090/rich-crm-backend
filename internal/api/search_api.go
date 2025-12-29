package api

import (
	"context"
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type SearchApi struct {
	Controller *controllers.SearchController
	config     *config.Config
}

func NewSearchApi(controller *controllers.SearchController, config *config.Config) Route {
	return &SearchApi{
		Controller: controller,
		config:     config,
	}
}

func (api *SearchApi) Setup(app *fiber.App) {
	api.Controller.Service.GlobalSearch(context.TODO(), "", [12]byte{}) // No-op to satisfy interface? No, just wiring.

	group := app.Group("/api/search", middleware.AuthMiddleware(api.config.SkipAuth))

	group.Get("/", api.Controller.GlobalSearch)
}
