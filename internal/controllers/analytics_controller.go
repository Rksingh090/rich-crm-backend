package controllers

import (
	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AnalyticsController struct {
	Service service.AnalyticsService
}

func NewAnalyticsController(service service.AnalyticsService) *AnalyticsController {
	return &AnalyticsController{Service: service}
}

// CreateMetric creates a new metric
// @Summary Create metric
// @Tags analytics
// @Accept json
// @Produce json
// @Param metric body models.Metric true "Metric"
// @Success 201 {object} models.Metric
// @Router /api/analytics/metrics [post]
func (c *AnalyticsController) CreateMetric(ctx *fiber.Ctx) error {
	var metric models.Metric
	if err := ctx.BodyParser(&metric); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Get user ID from context
	if userID, ok := ctx.Locals("user_id").(primitive.ObjectID); ok {
		metric.CreatedBy = userID
	}

	if err := c.Service.CreateMetric(ctx.Context(), &metric); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(metric)
}

// GetMetric retrieves a metric by ID
// @Summary Get metric
// @Tags analytics
// @Produce json
// @Param id path string true "Metric ID"
// @Success 200 {object} models.Metric
// @Router /api/analytics/metrics/{id} [get]
func (c *AnalyticsController) GetMetric(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	metric, err := c.Service.GetMetric(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Metric not found"})
	}

	return ctx.JSON(metric)
}

// ListMetrics lists all metrics
// @Summary List metrics
// @Tags analytics
// @Produce json
// @Success 200 {array} models.Metric
// @Router /api/analytics/metrics [get]
func (c *AnalyticsController) ListMetrics(ctx *fiber.Ctx) error {
	metrics, err := c.Service.ListMetrics(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(metrics)
}

// UpdateMetric updates a metric
// @Summary Update metric
// @Tags analytics
// @Accept json
// @Produce json
// @Param id path string true "Metric ID"
// @Param updates body map[string]interface{} true "Updates"
// @Success 200 {object} fiber.Map
// @Router /api/analytics/metrics/{id} [put]
func (c *AnalyticsController) UpdateMetric(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	var updates map[string]interface{}
	if err := ctx.BodyParser(&updates); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := c.Service.UpdateMetric(ctx.Context(), id, updates); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Metric updated successfully"})
}

// DeleteMetric deletes a metric
// @Summary Delete metric
// @Tags analytics
// @Param id path string true "Metric ID"
// @Success 200 {object} fiber.Map
// @Router /api/analytics/metrics/{id} [delete]
func (c *AnalyticsController) DeleteMetric(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	if err := c.Service.DeleteMetric(ctx.Context(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Metric deleted successfully"})
}

// CalculateMetric calculates a metric value
// @Summary Calculate metric
// @Tags analytics
// @Accept json
// @Produce json
// @Param id path string true "Metric ID"
// @Param filters body map[string]interface{} false "Additional Filters"
// @Success 200 {object} models.MetricResult
// @Router /api/analytics/metrics/{id}/calculate [post]
func (c *AnalyticsController) CalculateMetric(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	var filters map[string]interface{}
	ctx.BodyParser(&filters) // Optional filters

	result, err := c.Service.CalculateMetric(ctx.Context(), id, filters)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(result)
}

// GetMetricHistory retrieves metric history
// @Summary Get metric history
// @Tags analytics
// @Produce json
// @Param id path string true "Metric ID"
// @Param start query string false "Start time"
// @Param end query string false "End time"
// @Success 200 {array} models.MetricDataPoint
// @Router /api/analytics/metrics/{id}/history [get]
func (c *AnalyticsController) GetMetricHistory(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	// Parse time range from query params
	timeRange := models.TimeRange{
		// Start: parse start time
		// End: parse end time
	}

	history, err := c.Service.GetMetricHistory(ctx.Context(), id, timeRange)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(history)
}

// GetDashboardMetrics retrieves all metrics for a dashboard
// @Summary Get dashboard metrics
// @Tags analytics
// @Produce json
// @Param id path string true "Dashboard ID"
// @Success 200 {object} map[string]models.MetricResult
// @Router /api/analytics/dashboards/{id}/metrics [get]
func (c *AnalyticsController) GetDashboardMetrics(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	metrics, err := c.Service.GetDashboardMetrics(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(metrics)
}
