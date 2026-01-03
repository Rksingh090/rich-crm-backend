package sync

import (
	"github.com/gofiber/fiber/v2"
)

type SyncController struct {
	Service SyncService
}

func NewSyncController(service SyncService) *SyncController {
	return &SyncController{
		Service: service,
	}
}

// CreateSyncSetting godoc
// CreateSyncSetting godoc
// @Summary Create sync setting
// @Description Create a new synchronization configuration
// @Tags sync
// @Accept json
// @Produce json
// @Param setting body SyncSetting true "Sync Setting"
// @Success 201 {object} SyncSetting
// @Failure 400 {object} map[string]interface{}
// @Router /api/sync/settings [post]
func (ctrl *SyncController) CreateSyncSetting(c *fiber.Ctx) error {
	var setting SyncSetting
	if err := c.BodyParser(&setting); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.Service.CreateSetting(c.UserContext(), &setting); err != nil {
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
// ListSyncSettings godoc
// @Summary List sync settings
// @Description List all synchronization configurations
// @Tags sync
// @Produce json
// @Success 200 {array} SyncSetting
// @Failure 500 {object} map[string]interface{}
// @Router /api/sync/settings [get]
func (ctrl *SyncController) ListSyncSettings(c *fiber.Ctx) error {
	settings, err := ctrl.Service.ListSettings(c.UserContext())
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
// GetSyncSetting godoc
// @Summary Get sync setting
// @Description Get a sync configuration by ID
// @Tags sync
// @Produce json
// @Param id path string true "Setting ID"
// @Success 200 {object} SyncSetting
// @Failure 404 {object} map[string]interface{}
// @Router /api/sync/settings/{id} [get]
func (ctrl *SyncController) GetSyncSetting(c *fiber.Ctx) error {
	id := c.Params("id")

	setting, err := ctrl.Service.GetSetting(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(setting)
}

// UpdateSyncSetting godoc
// UpdateSyncSetting godoc
// @Summary Update sync setting
// @Description Update an existing sync configuration
// @Tags sync
// @Accept json
// @Produce json
// @Param id path string true "Setting ID"
// @Param setting body map[string]interface{} true "Sync Setting Updates"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/sync/settings/{id} [put]
func (ctrl *SyncController) UpdateSyncSetting(c *fiber.Ctx) error {
	id := c.Params("id")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.Service.UpdateSetting(c.UserContext(), id, updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Sync setting updated successfully",
	})
}

// DeleteSyncSetting godoc
// DeleteSyncSetting godoc
// @Summary Delete sync setting
// @Description Delete a sync configuration by ID
// @Tags sync
// @Param id path string true "Setting ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/sync/settings/{id} [delete]
func (ctrl *SyncController) DeleteSyncSetting(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.Service.DeleteSetting(c.UserContext(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Sync setting deleted successfully",
	})
}

// RunSync godoc
// RunSync godoc
// @Summary Run sync
// @Description Manually trigger a synchronization job
// @Tags sync
// @Produce json
// @Param id path string true "Setting ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/sync/settings/{id}/run [post]
func (ctrl *SyncController) RunSync(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.Service.RunSync(c.UserContext(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Sync job triggered successfully",
	})
}

// ListSyncLogs godoc
// ListSyncLogs godoc
// @Summary List sync logs
// @Description List logs for a specific sync setting
// @Tags sync
// @Produce json
// @Param id path string true "Setting ID"
// @Success 200 {array} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/sync/settings/{id}/logs [get]
func (ctrl *SyncController) ListSyncLogs(c *fiber.Ctx) error {
	id := c.Params("id")
	logs, err := ctrl.Service.ListLogs(c.UserContext(), id, 50)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": logs,
	})
}
