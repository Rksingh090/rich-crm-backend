package import_feature

import (
	"go-crm/internal/config"
	"go-crm/internal/features/role"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type ImportApi struct {
	ImportController *ImportController
	Config           *config.Config
	RoleService      role.RoleService
}

func NewImportApi(importController *ImportController, config *config.Config, roleService role.RoleService) *ImportApi {
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
