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
	app.Post("/upload", middleware.AuthMiddleware(h.config.SkipAuth), h.controller.UploadFile)

	// Static file serving
	app.Static("/uploads", "./uploads")
}
