package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"go-crm/internal/service"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ImportController struct {
	ImportService service.ImportService
	FileRepo      repository.FileRepository
	UploadDir     string
}

func NewImportController(importService service.ImportService, fileRepo repository.FileRepository) *ImportController {
	uploadDir := "./uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}
	return &ImportController{
		ImportService: importService,
		FileRepo:      fileRepo,
		UploadDir:     uploadDir,
	}
}

// UploadAndPreview godoc
// @Summary Upload file and preview import data
// @Tags Import
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to import (CSV or Excel)"
// @Param module formData string true "Module name"
// @Success 200 {object} models.ImportPreview
// @Router /api/import/preview [post]
func (c *ImportController) UploadAndPreview(ctx *fiber.Ctx) error {
	moduleName := ctx.FormValue("module")
	if moduleName == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "module is required"})
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}

	// Open file for preview
	file, err := fileHeader.Open()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to open file"})
	}
	defer file.Close()

	// Get preview
	preview, err := c.ImportService.PreviewFile(ctx.Context(), file, fileHeader.Filename, moduleName)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(preview)
}

// CreateImportJob godoc
// @Summary Create import job with file upload
// @Tags Import
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to import"
// @Param module formData string true "Module name"
// @Param mapping formData string true "Column mapping JSON"
// @Success 201 {object} models.ImportJob
// @Router /api/import/jobs [post]
func (c *ImportController) CreateImportJob(ctx *fiber.Ctx) error {
	moduleName := ctx.FormValue("module")
	mappingJSON := ctx.FormValue("mapping")

	if moduleName == "" || mappingJSON == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "module and mapping required"})
	}

	// Get user ID
	userIDStr, ok := ctx.Locals("userID").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found"})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	// Get file
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}

	// Save file using existing file system
	originalName := filepath.Base(fileHeader.Filename)
	timestamp := time.Now().UnixNano()
	uniqueName := fmt.Sprintf("%d_%s", timestamp, originalName)
	uniqueName = strings.ReplaceAll(uniqueName, " ", "_")
	dstPath := filepath.Join(c.UploadDir, uniqueName)

	if err := ctx.SaveFile(fileHeader, dstPath); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error saving file"})
	}

	// Save file metadata
	fileRecord := &models.File{
		OriginalFilename: originalName,
		UniqueFilename:   uniqueName,
		Path:             dstPath,
		URL:              fmt.Sprintf("/uploads/%s", uniqueName),
		Group:            "import",
		Size:             fileHeader.Size,
		MIMEType:         fileHeader.Header.Get("Content-Type"),
		CreatedAt:        time.Now(),
	}

	if err := c.FileRepo.Save(ctx.Context(), fileRecord); err != nil {
		os.Remove(dstPath) // Cleanup on error
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error saving file metadata"})
	}

	// Parse mapping JSON
	var mapping map[string]string
	if err := json.Unmarshal([]byte(mappingJSON), &mapping); err != nil {
		os.Remove(dstPath) // Cleanup on error
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid mapping JSON"})
	}

	// Get total rows from preview
	file, _ := fileHeader.Open()
	defer file.Close()
	preview, _ := c.ImportService.PreviewFile(ctx.Context(), file, fileHeader.Filename, moduleName)
	totalRows := 0
	if preview != nil {
		totalRows = preview.TotalRows
	}

	// Create import job
	job := &models.ImportJob{
		UserID:        userID,
		ModuleName:    moduleName,
		FileName:      originalName,
		FilePath:      dstPath,
		ColumnMapping: mapping,
		TotalRecords:  totalRows,
	}

	if err := c.ImportService.CreateJob(ctx.Context(), job); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(job)
}

// ExecuteImport godoc
// @Summary Execute import job
// @Tags Import
// @Param id path string true "Job ID"
// @Success 200
// @Router /api/import/jobs/{id}/execute [post]
func (c *ImportController) ExecuteImport(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	userIDStr, ok := ctx.Locals("userID").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found"})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	// Start import in background
	go func() {
		bgCtx := context.Background()
		c.ImportService.ProcessImport(bgCtx, id, userID)
	}()

	return ctx.JSON(fiber.Map{"message": "Import started"})
}

// GetImportJob godoc
// @Summary Get import job status
// @Tags Import
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} models.ImportJob
// @Router /api/import/jobs/{id} [get]
func (c *ImportController) GetImportJob(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	job, err := c.ImportService.GetJob(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Job not found"})
	}

	return ctx.JSON(job)
}

// ListImportJobs godoc
// @Summary List user's import jobs
// @Tags Import
// @Produce json
// @Success 200 {array} models.ImportJob
// @Router /api/import/jobs [get]
func (c *ImportController) ListImportJobs(ctx *fiber.Ctx) error {
	userIDStr, ok := ctx.Locals("userID").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found"})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	jobs, err := c.ImportService.GetUserJobs(ctx.Context(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(jobs)
}
