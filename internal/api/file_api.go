package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type FileApi struct {
	controller *controllers.FileController
	config     *config.Config
}

func NewFileApi(controller *controllers.FileController, config *config.Config) *FileApi {
	return &FileApi{
		controller: controller,
		config:     config,
	}
}

// Setup registers all file-related routes
func (h *FileApi) Setup(app *fiber.App) {
	// File upload route (protected)
	app.Post("/api/upload", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.UploadFile)

	// File management routes (protected)
	app.Get("/api/files/:module/:recordId", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.GetFilesByRecord)
	app.Get("/api/files/shared", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.GetSharedFiles)
	app.Get("/api/files/:id/download", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.DownloadFile)
	app.Delete("/api/files/:id", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.DeleteFile)

	// Static file serving - serve files from config FSPath via config FSURL
	app.Static(h.config.FSURL, h.config.FSPath)
}
