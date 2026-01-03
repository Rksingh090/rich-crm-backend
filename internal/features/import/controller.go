package import_feature

import (
	"context"
	"encoding/json"
	"fmt"
	"go-crm/internal/config"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ImportController struct {
	ImportService ImportService
	UploadDir     string
	Config        *config.Config
}

func NewImportController(importService ImportService, cfg *config.Config) *ImportController {
	if _, err := os.Stat(cfg.FSPath); os.IsNotExist(err) {
		os.MkdirAll(cfg.FSPath, 0755)
	}
	return &ImportController{
		ImportService: importService,
		UploadDir:     cfg.FSPath,
		Config:        cfg,
	}
}

// UploadAndPreview godoc
// @Summary Upload import file
// @Description Upload a CSV/Excel file and preview its content
// @Tags import
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Import File"
// @Param module formData string true "Module Name"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/import/upload [post]
func (c *ImportController) UploadAndPreview(ctx *fiber.Ctx) error {
	moduleName := ctx.FormValue("module")
	if moduleName == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "module is required"})
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to open file"})
	}
	defer file.Close()

	preview, err := c.ImportService.PreviewFile(ctx.UserContext(), file, fileHeader.Filename, moduleName)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(preview)
}

// CreateImportJob godoc
// @Summary Create import job
// @Description Create a new data import job
// @Tags import
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Import File"
// @Param module formData string true "Module Name"
// @Param mapping formData string true "Column Mapping JSON"
// @Success 201 {object} ImportJob
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/import/jobs [post]
func (c *ImportController) CreateImportJob(ctx *fiber.Ctx) error {
	moduleName := ctx.FormValue("module")
	mappingJSON := ctx.FormValue("mapping")

	if moduleName == "" || mappingJSON == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "module and mapping required"})
	}

	userIDStr, ok := ctx.Locals("user_id").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}

	originalName := filepath.Base(fileHeader.Filename)
	timestamp := time.Now().UnixNano()
	uniqueName := fmt.Sprintf("%d_%s", timestamp, originalName)
	uniqueName = strings.ReplaceAll(uniqueName, " ", "_")
	dstPath := filepath.Join(c.UploadDir, uniqueName)

	if err := ctx.SaveFile(fileHeader, dstPath); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error saving file"})
	}

	var mapping map[string]string
	if err := json.Unmarshal([]byte(mappingJSON), &mapping); err != nil {
		os.Remove(dstPath)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid mapping JSON"})
	}

	file, _ := fileHeader.Open()
	defer file.Close()
	preview, _ := c.ImportService.PreviewFile(ctx.UserContext(), file, fileHeader.Filename, moduleName)
	totalRows := 0
	if preview != nil {
		totalRows = preview.TotalRows
	}

	job := &ImportJob{
		UserID:        userID,
		ModuleName:    moduleName,
		FileName:      originalName,
		FilePath:      dstPath,
		ColumnMapping: mapping,
		TotalRecords:  totalRows,
	}

	if err := c.ImportService.CreateJob(ctx.UserContext(), job); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(job)
}

// ExecuteImport godoc
// @Summary Execute import job
// @Description Start processing an import job
// @Tags import
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/import/jobs/{id}/execute [post]
func (c *ImportController) ExecuteImport(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	userIDStr, ok := ctx.Locals("user_id").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	go func() {
		bgCtx := context.Background()
		c.ImportService.ProcessImport(bgCtx, id, userID)
	}()

	return ctx.JSON(fiber.Map{"message": "Import started"})
}

// GetImportJob godoc
// @Summary Get import job
// @Description Get details of an import job
// @Tags import
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} ImportJob
// @Failure 404 {object} map[string]interface{}
// @Router /api/import/jobs/{id} [get]
func (c *ImportController) GetImportJob(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	job, err := c.ImportService.GetJob(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Job not found"})
	}

	return ctx.JSON(job)
}

// ListImportJobs godoc
// @Summary List import jobs
// @Description List all import jobs for the current user
// @Tags import
// @Produce json
// @Success 200 {array} ImportJob
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/import/jobs [get]
func (c *ImportController) ListImportJobs(ctx *fiber.Ctx) error {
	userIDStr, ok := ctx.Locals("user_id").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	jobs, err := c.ImportService.GetUserJobs(ctx.UserContext(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(jobs)
}
