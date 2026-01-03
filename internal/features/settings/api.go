package settings

import (
	"go-crm/internal/common/api"
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type SettingsApi struct {
	Controller  *SettingsController
	Config      *config.Config
	RoleService middleware.RoleService
}

func NewSettingsApi(controller *SettingsController, config *config.Config, roleService middleware.RoleService) api.Route {
	return &SettingsApi{
		Controller:  controller,
		Config:      config,
		RoleService: roleService,
	}
}

func (a *SettingsApi) Setup(app *fiber.App) {
	group := app.Group("/api/settings", middleware.AuthMiddleware(a.Config.SkipAuth))

	group.Get("/email", middleware.RequirePermission(a.RoleService, "settings", "read"), a.Controller.GetEmailConfig)
	group.Put("/email", middleware.RequirePermission(a.RoleService, "settings", "update"), a.Controller.UpdateEmailConfig)
	group.Get("/general", middleware.RequirePermission(a.RoleService, "settings", "read"), a.Controller.GetGeneralConfig)
	group.Put("/general", middleware.RequirePermission(a.RoleService, "settings", "update"), a.Controller.UpdateGeneralConfig)

	// File Sharing Settings
	group.Get("/file-sharing", middleware.RequirePermission(a.RoleService, "settings", "read"), a.Controller.GetFileSharingConfig)
	group.Put("/file-sharing", middleware.RequirePermission(a.RoleService, "settings", "update"), a.Controller.UpdateFileSharingConfig)
}
