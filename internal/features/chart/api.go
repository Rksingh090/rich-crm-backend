package chart

import (
	"go-crm/internal/common/api"
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type ChartApi struct {
	ChartController *ChartController
	Config          *config.Config
	RoleService     middleware.RoleService
}

func NewChartApi(chartController *ChartController, cfg *config.Config, roleService middleware.RoleService) api.Route {
	return &ChartApi{
		ChartController: chartController,
		Config:          cfg,
		RoleService:     roleService,
	}
}

func (api *ChartApi) Setup(app *fiber.App) {
	group := app.Group("/api/charts", middleware.AuthMiddleware(api.Config.SkipAuth))

	group.Post("/", middleware.RequirePermission(api.RoleService, "charts", "create"), api.ChartController.Create)
	group.Get("/", middleware.RequirePermission(api.RoleService, "charts", "read"), api.ChartController.List)
	group.Get("/:id", middleware.RequirePermission(api.RoleService, "charts", "read"), api.ChartController.Get)
	group.Put("/:id", middleware.RequirePermission(api.RoleService, "charts", "update"), api.ChartController.Update)
	group.Delete("/:id", middleware.RequirePermission(api.RoleService, "charts", "delete"), api.ChartController.Delete)
	group.Get("/:id/data", middleware.RequirePermission(api.RoleService, "charts", "read"), api.ChartController.GetData)
}
