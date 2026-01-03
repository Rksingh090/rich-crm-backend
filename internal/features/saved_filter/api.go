package saved_filter

import (
	"go-crm/internal/config"
	"go-crm/internal/features/role"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type SavedFilterApi struct {
	FilterController *SavedFilterController
	Config           *config.Config
	RoleService      role.RoleService
}

func NewSavedFilterApi(filterController *SavedFilterController, config *config.Config, roleService role.RoleService) *SavedFilterApi {
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
