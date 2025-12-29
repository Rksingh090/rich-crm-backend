package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type SavedFilterApi struct {
	FilterController *controllers.SavedFilterController
	Config           *config.Config
	RoleService      service.RoleService
}

func NewSavedFilterApi(filterController *controllers.SavedFilterController, config *config.Config, roleService service.RoleService) Route {
	return &SavedFilterApi{
		FilterController: filterController,
		Config:           config,
		RoleService:      roleService,
	}
}

func (api *SavedFilterApi) Setup(app *fiber.App) {
	group := app.Group("/api/filters", middleware.AuthMiddleware(api.Config.SkipAuth))

	group.Post("/", api.FilterController.CreateFilter)
	group.Get("/", api.FilterController.ListUserFilters)
	group.Get("/public", api.FilterController.ListPublicFilters)
	group.Get("/:id", api.FilterController.GetFilter)
	group.Put("/:id", api.FilterController.UpdateFilter)
	group.Delete("/:id", api.FilterController.DeleteFilter)
}
