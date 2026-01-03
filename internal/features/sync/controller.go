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
func (ctrl *SyncController) CreateSyncSetting(c *fiber.Ctx) error {
	var setting SyncSetting
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
