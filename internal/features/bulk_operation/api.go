package bulk_operation

import (
	"go-crm/internal/config"
	"go-crm/internal/features/role"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type BulkOperationApi struct {
	BulkController *BulkOperationController
	Config         *config.Config
	RoleService    role.RoleService
}

func NewBulkOperationApi(bulkController *BulkOperationController, config *config.Config, roleService role.RoleService) *BulkOperationApi {
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
