package controllers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-crm/internal/config"
	"go-crm/internal/models"
	"go-crm/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type FileController struct {
	UploadDir string
	Repo      repository.FileRepository
}

func NewFileController(repo repository.FileRepository, cfg *config.Config) *FileController {
	// Ensure upload directory exists
	uploadDir := "./uploads" // Using a default, could be cfg.UploadDir if added to config
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}
	return &FileController{
		UploadDir: uploadDir,
		Repo:      repo,
	}
}

// UploadFile godoc
// @Summary      Upload a file
// @Description  Upload a file and get a URL (Metadata stored in DB)
// @Tags         files
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData file   true  "File to upload"
// @Param        group formData string false "Group (optional)"
// @Success      200   {object} models.File
// @Failure      400   {string} string "Invalid input"
// @Failure      500   {string} string "Internal Server Error"
// @Router       /upload [post]
func (ctrl *FileController) UploadFile(c *fiber.Ctx) error {
	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error retrieving file",
		})
	}

	// Check file size (10 MB limit)
	if file.Size > 10<<20 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File too large (max 10MB)",
		})
	}

	// Create unique filename
	originalName := filepath.Base(file.Filename)
	timestamp := time.Now().UnixNano()
	uniqueName := fmt.Sprintf("%d_%s", timestamp, originalName)
	uniqueName = strings.ReplaceAll(uniqueName, " ", "_")

	dstPath := filepath.Join(ctrl.UploadDir, uniqueName)

	// Save file to disk
	if err := c.SaveFile(file, dstPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error saving file to disk",
		})
	}

	// Save metadata to DB
	fileRecord := &models.File{
		OriginalFilename: originalName,
		UniqueFilename:   uniqueName,
		Path:             dstPath,
		URL:              fmt.Sprintf("/uploads/%s", uniqueName),
		Group:            c.FormValue("group"),
		Size:             file.Size,
		MIMEType:         file.Header.Get("Content-Type"),
		CreatedAt:        time.Now(),
	}

	if err := ctrl.Repo.Save(c.Context(), fileRecord); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error saving file metadata",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fileRecord)
}
