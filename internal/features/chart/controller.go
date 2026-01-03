package chart

import (
	"github.com/gofiber/fiber/v2"
)

type ChartController struct {
	ChartService ChartService
}

func NewChartController(chartService ChartService) *ChartController {
	return &ChartController{ChartService: chartService}
}

// Create godoc
// Create godoc
// @Summary Create chart
// @Description Create a new chart configuration
// @Tags charts
// @Accept json
// @Produce json
// @Param chart body Chart true "Chart Configuration"
// @Success 201 {object} Chart
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/charts [post]
func (c *ChartController) Create(ctx *fiber.Ctx) error {
	var ch Chart
	if err := ctx.BodyParser(&ch); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.ChartService.CreateChart(ctx.UserContext(), &ch); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(ch)
}

// List godoc
// List godoc
// @Summary List charts
// @Description List all available charts
// @Tags charts
// @Produce json
// @Success 200 {array} Chart
// @Failure 500 {object} map[string]interface{}
// @Router /api/charts [get]
func (c *ChartController) List(ctx *fiber.Ctx) error {
	charts, err := c.ChartService.ListCharts(ctx.UserContext())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(charts)
}

// Get godoc
// Get godoc
// @Summary Get chart
// @Description Get a chart configuration by ID
// @Tags charts
// @Produce json
// @Param id path string true "Chart ID"
// @Success 200 {object} Chart
// @Failure 404 {object} map[string]interface{}
// @Router /api/charts/{id} [get]
func (c *ChartController) Get(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	ch, err := c.ChartService.GetChart(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Chart not found"})
	}
	return ctx.JSON(ch)
}

// Update godoc
// Update godoc
// @Summary Update chart
// @Description Update an existing chart configuration
// @Tags charts
// @Accept json
// @Produce json
// @Param id path string true "Chart ID"
// @Param chart body Chart true "Chart Configuration"
// @Success 200 {object} Chart
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/charts/{id} [put]
func (c *ChartController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var ch Chart
	if err := ctx.BodyParser(&ch); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.ChartService.UpdateChart(ctx.UserContext(), id, &ch); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(ch)
}

// Delete godoc
// Delete godoc
// @Summary Delete chart
// @Description Delete a chart configuration
// @Tags charts
// @Param id path string true "Chart ID"
// @Success 204 {object} nil
// @Failure 500 {object} map[string]interface{}
// @Router /api/charts/{id} [delete]
func (c *ChartController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.ChartService.DeleteChart(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// GetData godoc
// GetData godoc
// @Summary Get chart data
// @Description Get the calculated data for a chart
// @Tags charts
// @Produce json
// @Param id path string true "Chart ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/charts/{id}/data [get]
func (c *ChartController) GetData(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	data, err := c.ChartService.GetChartData(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(data)
}
