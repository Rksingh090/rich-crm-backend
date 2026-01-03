package search

import (
	"go-crm/internal/common/api"
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type SearchApi struct {
	Controller *SearchController
	config     *config.Config
}

func NewSearchApi(controller *SearchController, config *config.Config) api.Route {
	return &SearchApi{
		Controller: controller,
		config:     config,
	}
}

func (api *SearchApi) Setup(app *fiber.App) {
	group := app.Group("/api/search", middleware.AuthMiddleware(api.config.SkipAuth))
	group.Get("/", api.Controller.GlobalSearch)
}
