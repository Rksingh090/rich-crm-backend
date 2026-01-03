package report

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReportController struct {
	ReportService ReportService
}

func NewReportController(reportService ReportService) *ReportController {
	return &ReportController{ReportService: reportService}
}

// Create godoc
func (c *ReportController) Create(ctx *fiber.Ctx) error {
	var report Report
	if err := ctx.BodyParser(&report); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.ReportService.CreateReport(ctx.Context(), &report); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(report)
}

// List godoc
func (c *ReportController) List(ctx *fiber.Ctx) error {
	reports, err := c.ReportService.ListReports(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(reports)
}

// Get godoc
func (c *ReportController) Get(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	report, err := c.ReportService.GetReport(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Report not found"})
	}
	return ctx.JSON(report)
}

// Update godoc
func (c *ReportController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var report Report
	if err := ctx.BodyParser(&report); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.ReportService.UpdateReport(ctx.Context(), id, &report); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(report)
}

// Delete godoc
func (c *ReportController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.ReportService.DeleteReport(ctx.Context(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// Run godoc
func (c *ReportController) Run(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

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

// Export godoc
func (c *ReportController) Export(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	format := ctx.Query("format", "csv")

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

// RunPivot godoc
func (c *ReportController) RunPivot(ctx *fiber.Ctx) error {
	var config PivotConfig
	if err := ctx.BodyParser(&config); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	moduleName := ctx.Query("module")
	if moduleName == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "module query parameter is required"})
	}

	userIDStr, ok := ctx.Locals("userID").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found"})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	filters := make(map[string]any)
	result, err := c.ReportService.RunPivotReport(ctx.Context(), &config, moduleName, filters, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(result)
}

// RunCrossModule godoc
func (c *ReportController) RunCrossModule(ctx *fiber.Ctx) error {
	var config CrossModuleConfig
	if err := ctx.BodyParser(&config); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	userIDStr, ok := ctx.Locals("userID").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found"})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	filters := make(map[string]any)
	result, err := c.ReportService.RunCrossModuleReport(ctx.Context(), &config, filters, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(result)
}

// ExportExcel godoc
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
