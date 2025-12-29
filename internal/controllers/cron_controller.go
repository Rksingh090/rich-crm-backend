package controllers

import (
	"context"
	"go-crm/internal/models"
	"go-crm/internal/service"
	"time"

	"github.com/gofiber/fiber/v2"
)

type CronController struct {
	Service service.CronService
}

func NewCronController(service service.CronService) *CronController {
	return &CronController{
		Service: service,
	}
}

// CreateCronJob godoc
// @Summary Create a new cron job
// @Tags cron
// @Accept json
// @Produce json
// @Param cronJob body models.CronJob true "Cron Job"
// @Success 201 {object} models.CronJob
// @Router /api/cron-jobs [post]
func (c *CronController) CreateCronJob(ctx *fiber.Ctx) error {
	var cronJob models.CronJob
	if err := ctx.BodyParser(&cronJob); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	ctxt, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.Service.CreateCronJob(ctxt, &cronJob); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(cronJob)
}

// ListCronJobs godoc
// @Summary List cron jobs
// @Tags cron
// @Produce json
// @Param active query boolean false "Filter by active status"
// @Param module_id query string false "Filter by module ID"
// @Success 200 {array} models.CronJob
// @Router /api/cron-jobs [get]
func (c *CronController) ListCronJobs(ctx *fiber.Ctx) error {
	filter := make(map[string]interface{})

	// Apply filters
	if active := ctx.Query("active"); active != "" {
		filter["active"] = active == "true"
	}

	if moduleID := ctx.Query("module_id"); moduleID != "" {
		filter["module_id"] = moduleID
	}

	ctxt, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cronJobs, err := c.Service.ListCronJobs(ctxt, filter)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(cronJobs)
}

// GetCronJob godoc
// @Summary Get cron job by ID
// @Tags cron
// @Produce json
// @Param id path string true "Cron Job ID"
// @Success 200 {object} models.CronJob
// @Failure 404 {string} string "Cron job not found"
// @Router /api/cron-jobs/{id} [get]
func (c *CronController) GetCronJob(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	ctxt, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cronJob, err := c.Service.GetCronJob(ctxt, id)
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
// @Tags cron
// @Accept json
// @Produce json
// @Param id path string true "Cron Job ID"
// @Param cronJob body models.CronJob true "Cron Job Update"
// @Success 200 {object} models.CronJob
// @Router /api/cron-jobs/{id} [put]
func (c *CronController) UpdateCronJob(ctx *fiber.Ctx) error {
	var cronJob models.CronJob
	if err := ctx.BodyParser(&cronJob); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	ctxt, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.Service.UpdateCronJob(ctxt, &cronJob); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(cronJob)
}

// DeleteCronJob godoc
// @Summary Delete cron job
// @Tags cron
// @Param id path string true "Cron Job ID"
// @Success 204
// @Router /api/cron-jobs/{id} [delete]
func (c *CronController) DeleteCronJob(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	ctxt, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.Service.DeleteCronJob(ctxt, id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

// ExecuteCronJob godoc
// @Summary Manually execute a cron job
// @Tags cron
// @Param id path string true "Cron Job ID"
// @Success 200 {object} map[string]string
// @Router /api/cron-jobs/{id}/execute [post]
func (c *CronController) ExecuteCronJob(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	ctxt, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := c.Service.ExecuteCronJob(ctxt, id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Cron job executed successfully"})
}

// GetCronJobLogs godoc
// @Summary Get execution logs for a cron job
// @Tags cron
// @Produce json
// @Param id path string true "Cron Job ID"
// @Param limit query int false "Limit number of logs" default(50)
// @Success 200 {array} models.CronJobLog
// @Router /api/cron-jobs/{id}/logs [get]
func (c *CronController) GetCronJobLogs(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	limit := ctx.QueryInt("limit", 50)

	ctxt, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logs, err := c.Service.GetCronJobLogs(ctxt, id, limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(logs)
}
