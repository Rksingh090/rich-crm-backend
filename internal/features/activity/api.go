package activity

import (
	"go-crm/internal/config"
	"go-crm/internal/features/role"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type ActivityApi struct {
	ActivityController *ActivityController
	RoleService        role.RoleService
	Config             *config.Config
}

func NewActivityApi(activityController *ActivityController, roleService role.RoleService, config *config.Config) *ActivityApi {
	return &ActivityApi{
		ActivityController: activityController,
		RoleService:        roleService,
		Config:             config,
	}
}

func (api *ActivityApi) Setup(app *fiber.App) {
	group := app.Group("/api/activities", middleware.AuthMiddleware(api.Config.SkipAuth))
	group.Get("/calendar", api.ActivityController.GetCalendarEvents)
}
