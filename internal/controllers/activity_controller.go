package controllers

import (
	"go-crm/internal/service"
	"time"

	"github.com/gofiber/fiber/v2"
)

type ActivityController struct {
	ActivityService service.ActivityService
}

func NewActivityController(activityService service.ActivityService) *ActivityController {
	return &ActivityController{ActivityService: activityService}
}

// GetCalendarEvents godoc
// @Summary Get calendar events (tasks, calls, meetings)
// @Description Fetch events within a date range
// @Tags activities
// @Accept json
// @Produce json
// @Param start query string true "Start Date (YYYY-MM-DD)"
// @Param end query string true "End Date (YYYY-MM-DD)"
// @Success 200 {array} map[string]interface{}
// @Router /api/activities/calendar [get]
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

	// Adjust end date to include the full day
	end = end.Add(24 * time.Hour)

	events, err := c.ActivityService.GetCalendarEvents(ctx.Context(), start, end)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch events"})
	}

	return ctx.JSON(events)
}
