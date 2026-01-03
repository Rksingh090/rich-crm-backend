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
// Create godoc
// @Summary Create report
// @Description Create a new report configuration
// @Tags reports
// @Accept json
// @Produce json
// @Param report body Report true "Report Config"
// @Success 201 {object} Report
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/reports [post]
func (c *ReportController) Create(ctx *fiber.Ctx) error {
	var report Report
	if err := ctx.BodyParser(&report); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.ReportService.CreateReport(ctx.UserContext(), &report); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(report)
}

// List godoc
// List godoc
// @Summary List reports
// @Description List all reports
// @Tags reports
// @Produce json
// @Success 200 {array} Report
// @Failure 500 {object} map[string]interface{}
// @Router /api/reports [get]
func (c *ReportController) List(ctx *fiber.Ctx) error {
	reports, err := c.ReportService.ListReports(ctx.UserContext())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(reports)
}

// Get godoc
// Get godoc
// @Summary Get report
// @Description Get a report by ID
// @Tags reports
// @Produce json
// @Param id path string true "Report ID"
// @Success 200 {object} Report
// @Failure 404 {object} map[string]interface{}
// @Router /api/reports/{id} [get]
func (c *ReportController) Get(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	report, err := c.ReportService.GetReport(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Report not found"})
	}
	return ctx.JSON(report)
}

// Update godoc
// Update godoc
// @Summary Update report
// @Description Update an existing report configuration
// @Tags reports
// @Accept json
// @Produce json
// @Param id path string true "Report ID"
// @Param report body Report true "Report Config"
// @Success 200 {object} Report
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/reports/{id} [put]
func (c *ReportController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var report Report
	if err := ctx.BodyParser(&report); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.ReportService.UpdateReport(ctx.UserContext(), id, &report); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(report)
}

// Delete godoc
// Delete godoc
// @Summary Delete report
// @Description Delete a report by ID
// @Tags reports
// @Param id path string true "Report ID"
// @Success 204 {object} nil
// @Failure 500 {object} map[string]interface{}
// @Router /api/reports/{id} [delete]
func (c *ReportController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.ReportService.DeleteReport(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// Run godoc
// Run godoc
// @Summary Run report
// @Description Execute a report and get the results
// @Tags reports
// @Produce json
// @Param id path string true "Report ID"
// @Success 200 {array} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/reports/{id}/run [post]
func (c *ReportController) Run(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	userIDStr, ok := ctx.Locals("user_id").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found"})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	data, err := c.ReportService.RunReport(ctx.UserContext(), id, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(data)
}

// Export godoc
// Export godoc
// @Summary Export report
// @Description Export report results to a file (CSV)
// @Tags reports
// @Produce text/csv
// @Param id path string true "Report ID"
// @Param format query string false "Export format (default: csv)"
// @Success 200 {file} file "Report file"
// @Failure 500 {object} map[string]interface{}
// @Router /api/reports/{id}/export [get]
func (c *ReportController) Export(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	format := ctx.Query("format", "csv")

	userIDStr, ok := ctx.Locals("user_id").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found"})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	data, filename, err := c.ReportService.ExportReport(ctx.UserContext(), id, format, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	ctx.Set("Content-Type", "text/csv")
	ctx.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	return ctx.Send(data)
}

// RunPivot godoc
// RunPivot godoc
// @Summary Run pivot report
// @Description Execute a pivot table report
// @Tags reports
// @Accept json
// @Produce json
// @Param module query string true "Module Name"
// @Param config body PivotConfig true "Pivot Configuration"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/reports/pivot [post]
func (c *ReportController) RunPivot(ctx *fiber.Ctx) error {
	var config PivotConfig
	if err := ctx.BodyParser(&config); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	moduleName := ctx.Query("module")
	if moduleName == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "module query parameter is required"})
	}

	userIDStr, ok := ctx.Locals("user_id").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found"})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	filters := make(map[string]any)
	result, err := c.ReportService.RunPivotReport(ctx.UserContext(), &config, moduleName, filters, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(result)
}

// RunCrossModule godoc
// RunCrossModule godoc
// @Summary Run cross-module report
// @Description Execute a report spanning multiple modules
// @Tags reports
// @Accept json
// @Produce json
// @Param config body CrossModuleConfig true "Cross-Module Configuration"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/reports/cross-module [post]
func (c *ReportController) RunCrossModule(ctx *fiber.Ctx) error {
	var config CrossModuleConfig
	if err := ctx.BodyParser(&config); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	userIDStr, ok := ctx.Locals("user_id").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found"})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	filters := make(map[string]any)
	result, err := c.ReportService.RunCrossModuleReport(ctx.UserContext(), &config, filters, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(result)
}

// ExportExcel godoc
// ExportExcel godoc
// @Summary Export to Excel
// @Description Export raw data to an Excel file
// @Tags reports
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param request body map[string]interface{} true "Excel Export Request (data, columns)"
// @Success 200 {file} file "Excel file"
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/reports/export/excel [post]
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

	data, filename, err := c.ReportService.ExportToExcel(ctx.UserContext(), request.Data, request.Columns, request.Filename)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	ctx.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	return ctx.Send(data)
}
