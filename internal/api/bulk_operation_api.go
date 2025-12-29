package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type BulkOperationApi struct {
	BulkController *controllers.BulkOperationController
	Config         *config.Config
	RoleService    service.RoleService
}

func NewBulkOperationApi(bulkController *controllers.BulkOperationController, config *config.Config, roleService service.RoleService) Route {
	return &BulkOperationApi{
		BulkController: bulkController,
		Config:         config,
		RoleService:    roleService,
	}
}

func (api *BulkOperationApi) Setup(app *fiber.App) {
	group := app.Group("/api/bulk", middleware.AuthMiddleware(api.Config.SkipAuth))

	group.Post("/preview", api.BulkController.PreviewBulkOperation)
	group.Post("/operations", api.BulkController.CreateBulkOperation)
	group.Get("/operations", api.BulkController.ListBulkOperations)
	group.Get("/operations/:id", api.BulkController.GetBulkOperation)
	group.Post("/operations/:id/execute", api.BulkController.ExecuteBulkOperation)
}
