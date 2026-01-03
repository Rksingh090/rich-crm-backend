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
func (c *ChartController) Create(ctx *fiber.Ctx) error {
	var ch Chart
	if err := ctx.BodyParser(&ch); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.ChartService.CreateChart(ctx.Context(), &ch); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(ch)
}

// List godoc
func (c *ChartController) List(ctx *fiber.Ctx) error {
	charts, err := c.ChartService.ListCharts(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(charts)
}

// Get godoc
func (c *ChartController) Get(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	ch, err := c.ChartService.GetChart(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Chart not found"})
	}
	return ctx.JSON(ch)
}

// Update godoc
func (c *ChartController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var ch Chart
	if err := ctx.BodyParser(&ch); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.ChartService.UpdateChart(ctx.Context(), id, &ch); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(ch)
}

// Delete godoc
func (c *ChartController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.ChartService.DeleteChart(ctx.Context(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// GetData godoc
func (c *ChartController) GetData(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	data, err := c.ChartService.GetChartData(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(data)
}
