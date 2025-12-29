package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ImportApi struct {
	ImportController *controllers.ImportController
	Config           *config.Config
	RoleService      service.RoleService
}

func NewImportApi(importController *controllers.ImportController, config *config.Config, roleService service.RoleService) Route {
	return &ImportApi{
		ImportController: importController,
		Config:           config,
		RoleService:      roleService,
	}
}

func (api *ImportApi) Setup(app *fiber.App) {
	group := app.Group("/api/import", middleware.AuthMiddleware(api.Config.SkipAuth))

	group.Post("/preview", api.ImportController.UploadAndPreview)
	group.Post("/jobs", api.ImportController.CreateImportJob)
	group.Get("/jobs", api.ImportController.ListImportJobs)
	group.Get("/jobs/:id", api.ImportController.GetImportJob)
	group.Post("/jobs/:id/execute", api.ImportController.ExecuteImport)
}
