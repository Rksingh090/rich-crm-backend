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

// GetCalendarEvents godoc
// @Summary      Get calendar events
// @Description  Retrieve activity events within a specific date range
// @Tags         activity
// @Accept       json
// @Produce      json
// @Param        start  query     string  true  "Start date (YYYY-MM-DD)"
// @Param        end    query     string  true  "End date (YYYY-MM-DD)"
// @Success      200    {array}   any
// @Failure      400    {object}  map[string]string
// @Router       /api/activities/calendar [get]
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

// GetTimeline godoc
// @Summary      Get activity timeline
// @Description  Retrieve chronological timeline of activities (calls, meetings, tasks) for a record
// @Tags         activity
// @Produce      json
// @Param        module query     string  true  "Module Name (e.g. contacts)"
// @Param        id     query     string  true  "Record ID"
// @Success      200    {array}   any
// @Router       /api/activities/timeline [get]
func (c *ActivityController) GetTimeline(ctx *fiber.Ctx) error {
	moduleName := ctx.Query("module")
	recordID := ctx.Query("id")

	if moduleName == "" || recordID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "module and id are required"})
	}

	timeline, err := c.ActivityService.GetTimeline(ctx.UserContext(), moduleName, recordID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(timeline)
}
