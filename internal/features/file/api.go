package file

import (
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type FileApi struct {
	controller *FileController
	config     *config.Config
}

func NewFileApi(controller *FileController, config *config.Config) *FileApi {
	return &FileApi{
		controller: controller,
		config:     config,
	}
}

func (h *FileApi) Setup(app *fiber.App) {
	app.Post("/api/upload", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.UploadFile)
	app.Get("/api/files/:module/:recordId", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.GetFilesByRecord)
	app.Get("/api/files/shared", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.GetSharedFiles)
	app.Get("/api/files/:id/download", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.DownloadFile)
	app.Delete("/api/files/:id", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.DeleteFile)

	app.Static(h.config.FSURL, h.config.FSPath)
}
