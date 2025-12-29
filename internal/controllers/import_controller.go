package controllers

import (
	"context"
	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ImportController struct {
	ImportService service.ImportService
}

func NewImportController(importService service.ImportService) *ImportController {
	return &ImportController{
		ImportService: importService,
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
	// Get module name
	moduleName := ctx.FormValue("module")
	if moduleName == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "module is required"})
	}

	// Get uploaded file
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}

	// Open file
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
// @Summary Create an import job
// @Tags Import
// @Accept json
// @Produce json
// @Param job body models.ImportJob true "Import Job"
// @Success 201 {object} models.ImportJob
// @Router /api/import/jobs [post]
func (c *ImportController) CreateImportJob(ctx *fiber.Ctx) error {
	var job models.ImportJob
	if err := ctx.BodyParser(&job); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Get user ID from context
	userIDStr, ok := ctx.Locals("userID").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found"})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	job.UserID = userID

	if err := c.ImportService.CreateJob(ctx.Context(), &job); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(job)
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
	// Get user ID from context
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

// ExecuteImport godoc
// @Summary Execute import job
// @Tags Import
// @Param id path string true "Job ID"
// @Success 200
// @Router /api/import/jobs/{id}/execute [post]
func (c *ImportController) ExecuteImport(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	// Get user ID from context
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
		ctx := context.Background()
		c.ImportService.ProcessImport(ctx, id, userID)
	}()

	return ctx.JSON(fiber.Map{"message": "Import started"})
}
