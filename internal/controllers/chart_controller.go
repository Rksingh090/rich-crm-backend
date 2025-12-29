package controllers

import (
	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ChartController struct {
	ChartService service.ChartService
}

func NewChartController(chartService service.ChartService) *ChartController {
	return &ChartController{ChartService: chartService}
}

// CreateChart godoc
// @Summary Create a new chart
// @Tags Charts
// @Accept json
// @Produce json
// @Param chart body models.Chart true "Chart Definition"
// @Success 201 {object} models.Chart
// @Router /api/charts [post]
func (c *ChartController) Create(ctx *fiber.Ctx) error {
	var chart models.Chart
	if err := ctx.BodyParser(&chart); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.ChartService.CreateChart(ctx.Context(), &chart); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(chart)
}

// ListCharts godoc
// @Summary List all charts
// @Tags Charts
// @Produce json
// @Success 200 {array} models.Chart
// @Router /api/charts [get]
func (c *ChartController) List(ctx *fiber.Ctx) error {
	charts, err := c.ChartService.ListCharts(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(charts)
}

// GetChart godoc
// @Summary Get a chart by ID
// @Tags Charts
// @Produce json
// @Param id path string true "Chart ID"
// @Success 200 {object} models.Chart
// @Router /api/charts/{id} [get]
func (c *ChartController) Get(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	chart, err := c.ChartService.GetChart(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Chart not found"})
	}
	return ctx.JSON(chart)
}

// UpdateChart godoc
// @Summary Update a chart
// @Tags Charts
// @Accept json
// @Produce json
// @Param id path string true "Chart ID"
// @Param chart body models.Chart true "Chart Update"
// @Success 200 {object} models.Chart
// @Router /api/charts/{id} [put]
func (c *ChartController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var chart models.Chart
	if err := ctx.BodyParser(&chart); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.ChartService.UpdateChart(ctx.Context(), id, &chart); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(chart)
}

// DeleteChart godoc
// @Summary Delete a chart
// @Tags Charts
// @Param id path string true "Chart ID"
// @Success 204
// @Router /api/charts/{id} [delete]
func (c *ChartController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.ChartService.DeleteChart(ctx.Context(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// GetChartData godoc
// @Summary Get chart data
// @Tags Charts
// @Produce json
// @Param id path string true "Chart ID"
// @Success 200 {array} object
// @Router /api/charts/{id}/data [get]
func (c *ChartController) GetData(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	data, err := c.ChartService.GetChartData(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(data)
}
