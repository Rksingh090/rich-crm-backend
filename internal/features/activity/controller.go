package activity

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type ActivityController struct {
	ActivityService ActivityService
}

func NewActivityController(activityService ActivityService) *ActivityController {
	return &ActivityController{ActivityService: activityService}
}

func (c *ActivityController) GetCalendarEvents(ctx *fiber.Ctx) error {
	startStr := ctx.Query("start")
	endStr := ctx.Query("end")

	if startStr == "" || endStr == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Start and End dates are required"})
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid start date format (YYYY-MM-DD)"})
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid end date format (YYYY-MM-DD)"})
	}

	end = end.Add(24 * time.Hour)

	events, err := c.ActivityService.GetCalendarEvents(ctx.UserContext(), start, end)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch events"})
	}

	return ctx.JSON(events)
}
