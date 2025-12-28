package controllers

import (
	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type SyncController struct {
	Service service.SyncService
}

func NewSyncController(service service.SyncService) *SyncController {
	return &SyncController{
		Service: service,
	}
}

// CreateSyncSetting godoc
// @Summary      Create a new sync setting
// @Description  Register a new external DB sync setting
// @Tags         sync
// @Accept       json
// @Produce      json
// @Param        input body models.SyncSetting true "Sync Setting Data"
// @Success      201  {object} map[string]interface{}
// @Router       /api/sync/settings [post]
func (ctrl *SyncController) CreateSyncSetting(c *fiber.Ctx) error {
	var setting models.SyncSetting
	if err := c.BodyParser(&setting); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.Service.CreateSetting(c.Context(), &setting); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Sync setting created successfully",
		"data":    setting,
	})
}

// ListSyncSettings godoc
// @Summary      List all sync settings
// @Description  Get a list of all data sync settings
// @Tags         sync
// @Produce      json
// @Success      200 {array} models.SyncSetting
// @Router       /api/sync/settings [get]
func (ctrl *SyncController) ListSyncSettings(c *fiber.Ctx) error {
	settings, err := ctrl.Service.ListSettings(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": settings,
	})
}

// GetSyncSetting godoc
// @Summary      Get a sync setting
// @Description  Get a sync setting by ID
// @Tags         sync
// @Produce      json
// @Param        id path string true "Sync Setting ID"
// @Success      200 {object} models.SyncSetting
// @Router       /api/sync/settings/{id} [get]
func (ctrl *SyncController) GetSyncSetting(c *fiber.Ctx) error {
	id := c.Params("id")

	setting, err := ctrl.Service.GetSetting(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(setting)
}

// UpdateSyncSetting godoc
// @Summary      Update a sync setting
// @Description  Update a sync setting by ID
// @Tags         sync
// @Accept       json
// @Produce      json
// @Param        id    path string true "Sync Setting ID"
// @Param        input body map[string]interface{} true "Update Data"
// @Success      200  {object} map[string]string
// @Router       /api/sync/settings/{id} [put]
func (ctrl *SyncController) UpdateSyncSetting(c *fiber.Ctx) error {
	id := c.Params("id")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.Service.UpdateSetting(c.Context(), id, updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Sync setting updated successfully",
	})
}

// DeleteSyncSetting godoc
// @Summary      Delete a sync setting
// @Description  Delete a sync setting by ID
// @Tags         sync
// @Produce      json
// @Param        id path string true "Sync Setting ID"
// @Success      200 {object} map[string]string
// @Router       /api/sync/settings/{id} [delete]
func (ctrl *SyncController) DeleteSyncSetting(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.Service.DeleteSetting(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Sync setting deleted successfully",
	})
}

// RunSync godoc
// @Summary      Run sync manually
// @Description  Trigger a sync job manually for a setting
// @Tags         sync
// @Produce      json
// @Param        id path string true "Sync Setting ID"
// @Success      200 {object} map[string]string
// @Router       /api/sync/settings/{id}/run [post]
func (ctrl *SyncController) RunSync(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.Service.RunSync(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Sync job triggered successfully",
	})
}

// ListSyncLogs godoc
// @Summary      List sync logs
// @Description  Get history of sync jobs for a setting
// @Tags         sync
// @Produce      json
// @Param        id path string true "Sync Setting ID"
// @Success      200 {array} models.SyncLog
// @Router       /api/sync/settings/{id}/logs [get]
func (ctrl *SyncController) ListSyncLogs(c *fiber.Ctx) error {
	id := c.Params("id")
	logs, err := ctrl.Service.ListLogs(c.Context(), id, 50)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": logs,
	})
}
