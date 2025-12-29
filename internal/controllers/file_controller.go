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
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FileController struct {
	UploadDir   string
	Repo        repository.FileRepository
	FileService service.FileService
}

func NewFileController(repo repository.FileRepository, fileService service.FileService, cfg *config.Config) *FileController {
	// Ensure upload directory exists
	uploadDir := "./uploads" // Using a default, could be cfg.UploadDir if added to config
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}
	return &FileController{
		UploadDir:   uploadDir,
		Repo:        repo,
		FileService: fileService,
	}
}

// UploadFile godoc
// @Summary      Upload a file
// @Description  Upload a file and get a URL (Metadata stored in DB)
// @Tags         files
// @Accept       multipart/form-data
// @Produce      json
// @Param        file         formData file   true  "File to upload"
// @Param        module_name  formData string false "Module name (optional)"
// @Param        record_id    formData string false "Record ID (optional)"
// @Param        is_shared    formData bool   false "Is shared document (optional)"
// @Param        description  formData string false "File description (optional)"
// @Success      200   {object} models.File
// @Failure      400   {string} string "Invalid input"
// @Failure      500   {string} string "Internal Server Error"
// @Router       /upload [post]
func (ctrl *FileController) UploadFile(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("user_id").(primitive.ObjectID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error retrieving file",
		})
	}

	// Get optional parameters
	moduleName := c.FormValue("module_name")
	recordID := c.FormValue("record_id")
	isShared := c.FormValue("is_shared") == "true"
	description := c.FormValue("description")

	// Validate upload using service
	if err := ctrl.FileService.ValidateUpload(c.Context(), moduleName, recordID, file.Size, file.Header.Get("Content-Type")); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
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
		Size:             file.Size,
		MIMEType:         file.Header.Get("Content-Type"),
		ModuleName:       moduleName,
		RecordID:         recordID,
		UploadedBy:       userID,
		IsShared:         isShared,
		Description:      description,
		CreatedAt:        time.Now(),
	}

	if err := ctrl.Repo.Save(c.Context(), fileRecord); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error saving file metadata",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fileRecord)
}

// GetFilesByRecord godoc
// @Summary      Get files by record
// @Description  Get all files attached to a specific record
// @Tags         files
// @Produce      json
// @Param        module   path string true "Module name"
// @Param        recordId path string true "Record ID"
// @Success      200 {array} models.File
// @Router       /files/{module}/{recordId} [get]
func (ctrl *FileController) GetFilesByRecord(c *fiber.Ctx) error {
	moduleName := c.Params("module")
	recordID := c.Params("recordId")

	files, err := ctrl.FileService.GetFilesByRecord(c.Context(), moduleName, recordID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error retrieving files",
		})
	}

	return c.JSON(files)
}

// GetSharedFiles godoc
// @Summary      Get shared files
// @Description  Get all organization-wide shared documents
// @Tags         files
// @Produce      json
// @Success      200 {array} models.File
// @Router       /files/shared [get]
func (ctrl *FileController) GetSharedFiles(c *fiber.Ctx) error {
	files, err := ctrl.FileService.GetSharedFiles(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error retrieving shared files",
		})
	}

	return c.JSON(files)
}

// DownloadFile godoc
// @Summary      Download a file
// @Description  Download a file by ID
// @Tags         files
// @Produce      octet-stream
// @Param        id path string true "File ID"
// @Success      200
// @Router       /files/{id}/download [get]
func (ctrl *FileController) DownloadFile(c *fiber.Ctx) error {
	fileID := c.Params("id")

	file, err := ctrl.FileService.GetFile(c.Context(), fileID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	return c.Download(file.Path, file.OriginalFilename)
}

// DeleteFile godoc
// @Summary      Delete a file
// @Description  Delete a file by ID
// @Tags         files
// @Param        id path string true "File ID"
// @Success      200 {object} map[string]string
// @Router       /files/{id} [delete]
func (ctrl *FileController) DeleteFile(c *fiber.Ctx) error {
	fileID := c.Params("id")

	// Get user ID from context
	userID, ok := c.Locals("user_id").(primitive.ObjectID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if err := ctrl.FileService.DeleteFile(c.Context(), fileID, userID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "File deleted successfully",
	})
}
