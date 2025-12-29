package controllers

import (
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReportController struct {
	ReportService service.ReportService
}

func NewReportController(reportService service.ReportService) *ReportController {
	return &ReportController{ReportService: reportService}
}

// CreateReport godoc
// @Summary Create a new report
// @Tags Reports
// @Accept json
// @Produce json
// @Param report body models.Report true "Report Definition"
// @Success 201 {object} models.Report
// @Router /api/reports [post]
func (c *ReportController) Create(ctx *fiber.Ctx) error {
	var report models.Report
	if err := ctx.BodyParser(&report); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.ReportService.CreateReport(ctx.Context(), &report); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(report)
}

// ListReports godoc
// @Summary List all reports
// @Tags Reports
// @Produce json
// @Success 200 {array} models.Report
// @Router /api/reports [get]
func (c *ReportController) List(ctx *fiber.Ctx) error {
	reports, err := c.ReportService.ListReports(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(reports)
}

// GetReport godoc
// @Summary Get a report by ID
// @Tags Reports
// @Produce json
// @Param id path string true "Report ID"
// @Success 200 {object} models.Report
// @Router /api/reports/{id} [get]
func (c *ReportController) Get(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	report, err := c.ReportService.GetReport(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Report not found"})
	}
	return ctx.JSON(report)
}

// UpdateReport godoc
// @Summary Update a report
// @Tags Reports
// @Accept json
// @Produce json
// @Param id path string true "Report ID"
// @Param report body models.Report true "Report Update"
// @Success 200 {object} models.Report
// @Router /api/reports/{id} [put]
func (c *ReportController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var report models.Report
	if err := ctx.BodyParser(&report); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.ReportService.UpdateReport(ctx.Context(), id, &report); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(report)
}

// DeleteReport godoc
// @Summary Delete a report
// @Tags Reports
// @Param id path string true "Report ID"
// @Success 204
// @Router /api/reports/{id} [delete]
func (c *ReportController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.ReportService.DeleteReport(ctx.Context(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// RunReport godoc
// @Summary Run a report and return JSON data
// @Tags Reports
// @Produce json
// @Param id path string true "Report ID"
// @Success 200 {array} object
// @Router /api/reports/{id}/run [get]
func (c *ReportController) Run(ctx *fiber.Ctx) error {
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

	data, err := c.ReportService.RunReport(ctx.Context(), id, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(data)
}

// ExportReport godoc
// @Summary Export a report to CSV
// @Tags Reports
// @Produce text/csv
// @Param id path string true "Report ID"
// @Param format query string false "Format (default: csv)"
// @Success 200 {file} file
// @Router /api/reports/{id}/export [get]
func (c *ReportController) Export(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	format := ctx.Query("format", "csv")

	// Get user ID from context
	userIDStr, ok := ctx.Locals("userID").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found"})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	data, filename, err := c.ReportService.ExportReport(ctx.Context(), id, format, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	ctx.Set("Content-Type", "text/csv")
	ctx.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	return ctx.Send(data)
}

// RunPivotReport godoc
// @Summary Run a pivot table report
// @Tags Reports
// @Accept json
// @Produce json
// @Param request body models.PivotConfig true "Pivot Configuration"
// @Param module query string true "Module Name"
// @Success 200 {object} object
// @Router /api/reports/pivot [post]
func (c *ReportController) RunPivot(ctx *fiber.Ctx) error {
	var config models.PivotConfig
	if err := ctx.BodyParser(&config); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	moduleName := ctx.Query("module")
	if moduleName == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "module query parameter is required"})
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

	// Parse filters from query
	filters := make(map[string]any)

	result, err := c.ReportService.RunPivotReport(ctx.Context(), &config, moduleName, filters, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(result)
}

// RunCrossModuleReport godoc
// @Summary Run a cross-module report
// @Tags Reports
// @Accept json
// @Produce json
// @Param request body models.CrossModuleConfig true "Cross Module Configuration"
// @Success 200 {array} object
// @Router /api/reports/cross-module [post]
func (c *ReportController) RunCrossModule(ctx *fiber.Ctx) error {
	var config models.CrossModuleConfig
	if err := ctx.BodyParser(&config); err != nil {
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

	// Parse filters from query
	filters := make(map[string]any)

	result, err := c.ReportService.RunCrossModuleReport(ctx.Context(), &config, filters, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(result)
}

// ExportToExcel godoc
// @Summary Export data to Excel
// @Tags Reports
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param request body object true "Export Request"
// @Success 200 {file} file
// @Router /api/reports/export-excel [post]
func (c *ReportController) ExportExcel(ctx *fiber.Ctx) error {
	var request struct {
		Data     []map[string]any `json:"data"`
		Columns  []string         `json:"columns"`
		Filename string           `json:"filename"`
	}

	if err := ctx.BodyParser(&request); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if request.Filename == "" {
		request.Filename = fmt.Sprintf("export_%d", int64(primitive.NewObjectID().Timestamp().Unix()))
	}

	data, filename, err := c.ReportService.ExportToExcel(ctx.Context(), request.Data, request.Columns, request.Filename)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	ctx.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	return ctx.Send(data)
}
