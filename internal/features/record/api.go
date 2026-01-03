package record

import (
	"go-crm/internal/config"
	"go-crm/internal/features/role"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type RecordApi struct {
	recordController *RecordController
	config           *config.Config
	roleService      role.RoleService
}

func NewRecordApi(
	recordController *RecordController,
	config *config.Config,
	roleService role.RoleService,
) *RecordApi {
	return &RecordApi{
		recordController: recordController,
		config:           config,
		roleService:      roleService,
	}
}

// Setup registers record-related routes
func (h *RecordApi) Setup(app *fiber.App) {
	// Group is same as module Schema, but handles records
	modules := app.Group("/api/modules", middleware.AuthMiddleware(h.config.SkipAuth))

	modules.Get("/:name/records", h.recordController.ListRecords)
	modules.Post("/:name/records", h.recordController.CreateRecord)
	modules.Get("/:name/records/:id", h.recordController.GetRecord)
	modules.Put("/:name/records/:id", h.recordController.UpdateRecord)
	modules.Delete("/:name/records/:id", h.recordController.DeleteRecord)
}
