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
	// Register more specific routes first to avoid conflicts
	// Public static file access (from .env FS_URL)
	app.Static(h.config.FSURL, h.config.FSPath)

	// Download route (also public)
	app.Get("/api/files/download/:id", h.controller.DownloadFile)

	// Authenticated routes
	app.Post("/api/files/upload", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.UploadFile)
	app.Get("/api/files/shared", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.GetSharedFiles)
	app.Get("/api/files/:module/:recordId", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.GetFilesByRecord)
	app.Delete("/api/files/:id", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.DeleteFile)
}
