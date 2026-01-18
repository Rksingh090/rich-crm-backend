package cron_feature

import (
	"github.com/gofiber/fiber/v2"
)

type CronController struct {
	Service CronService
}

func NewCronController(service CronService) *CronController {
	return &CronController{
		Service: service,
	}
}

// CreateCronJob godoc
// @Summary Create cron job
// @Description Create a new cron job
// @Tags cron
// @Accept json
// @Produce json
// @Param job body CronJob true "Cron Job"
// @Success 201 {object} CronJob
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/cron/jobs [post]
func (c *CronController) CreateCronJob(ctx *fiber.Ctx) error {
	var cronJob CronJob
	if err := ctx.BodyParser(&cronJob); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.Service.CreateCronJob(ctx.UserContext(), &cronJob); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(cronJob)
}

// ListCronJobs godoc
// @Summary List cron jobs
// @Description List all cron jobs with optional filtering
// @Tags cron
// @Produce json
// @Param active query boolean false "Filter by active status"
// @Param module_id query string false "Filter by module ID"
// @Success 200 {array} CronJob
// @Failure 500 {object} map[string]any
// @Router /api/cron/jobs [get]
func (c *CronController) ListCronJobs(ctx *fiber.Ctx) error {
	filter := make(map[string]any)

	if active := ctx.Query("active"); active != "" {
		filter["active"] = active == "true"
	}

	if moduleID := ctx.Query("module_id"); moduleID != "" {
		filter["module_id"] = moduleID
	}

	cronJobs, err := c.Service.ListCronJobs(ctx.UserContext(), filter)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(cronJobs)
}

// GetCronJob godoc
// @Summary Get cron job
// @Description Get a cron job by ID
// @Tags cron
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} CronJob
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/cron/jobs/{id} [get]
func (c *CronController) GetCronJob(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	cronJob, err := c.Service.GetCronJob(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if cronJob == nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Cron job not found"})
	}

	return ctx.JSON(cronJob)
}

// UpdateCronJob godoc
// @Summary Update cron job
// @Description Update an existing cron job
// @Tags cron
// @Accept json
// @Produce json
// @Param id path string true "Job ID"
// @Param job body CronJob true "Cron Job"
// @Success 200 {object} CronJob
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/cron/jobs/{id} [put]
func (c *CronController) UpdateCronJob(ctx *fiber.Ctx) error {
	var cronJob CronJob
	if err := ctx.BodyParser(&cronJob); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.Service.UpdateCronJob(ctx.UserContext(), &cronJob); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(cronJob)
}

// DeleteCronJob godoc
// @Summary Delete cron job
// @Description Delete a cron job by ID
// @Tags cron
// @Param id path string true "Job ID"
// @Success 204 {object} nil
// @Failure 500 {object} map[string]any
// @Router /api/cron/jobs/{id} [delete]
func (c *CronController) DeleteCronJob(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	if err := c.Service.DeleteCronJob(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

// ExecuteCronJob godoc
// @Summary Execute cron job
// @Description Manually trigger a cron job execution
// @Tags cron
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/cron/jobs/{id}/execute [post]
func (c *CronController) ExecuteCronJob(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	if err := c.Service.ExecuteCronJob(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Cron job executed successfully"})
}

// GetCronJobLogs godoc
// @Summary Get cron job logs
// @Description Get execution logs for a cron job
// @Tags cron
// @Produce json
// @Param id path string true "Job ID"
// @Param limit query int false "Max logs to return"
// @Success 200 {array} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/cron/jobs/{id}/logs [get]
func (c *CronController) GetCronJobLogs(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	limit := ctx.QueryInt("limit", 50)

	logs, err := c.Service.GetCronJobLogs(ctx.UserContext(), id, limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(logs)
}
