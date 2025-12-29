package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ChartApi struct {
	ChartController *controllers.ChartController
	Config          *config.Config
	RoleService     service.RoleService
}

func NewChartApi(chartController *controllers.ChartController, config *config.Config, roleService service.RoleService) Route {
	return &ChartApi{
		ChartController: chartController,
		Config:          config,
		RoleService:     roleService,
	}
}

func (api *ChartApi) Setup(app *fiber.App) {
	group := app.Group("/api/charts", middleware.AuthMiddleware(api.Config.SkipAuth))

	// Using "charts" resource for permissions
	group.Post("/", middleware.RequirePermission(api.RoleService, "charts", "create"), api.ChartController.Create)
	group.Get("/", middleware.RequirePermission(api.RoleService, "charts", "read"), api.ChartController.List)
	group.Get("/:id", middleware.RequirePermission(api.RoleService, "charts", "read"), api.ChartController.Get)
	group.Put("/:id", middleware.RequirePermission(api.RoleService, "charts", "update"), api.ChartController.Update)
	group.Delete("/:id", middleware.RequirePermission(api.RoleService, "charts", "delete"), api.ChartController.Delete)
	group.Get("/:id/data", middleware.RequirePermission(api.RoleService, "charts", "read"), api.ChartController.GetData)
}
