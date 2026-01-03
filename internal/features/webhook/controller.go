package webhook

import (
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WebhookController struct {
	Service WebhookService
}

func NewWebhookController(service WebhookService) *WebhookController {
	return &WebhookController{
		Service: service,
	}
}

// CreateWebhook godoc
// CreateWebhook godoc
// @Summary Create webhook
// @Description Create a new webhook
// @Tags webhooks
// @Accept json
// @Produce json
// @Param webhook body Webhook true "Webhook Details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/webhooks [post]
func (ctrl *WebhookController) CreateWebhook(c *fiber.Ctx) error {
	var webhook Webhook
	if err := c.BodyParser(&webhook); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userIDStr, ok := c.Locals("user_id").(string)
	if ok {
		if oid, err := primitive.ObjectIDFromHex(userIDStr); err == nil {
			webhook.CreatedBy = oid
		}
	}

	if err := ctrl.Service.CreateWebhook(c.UserContext(), &webhook); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Webhook created successfully",
		"data":    webhook,
	})
}

// ListWebhooks godoc
// ListWebhooks godoc
// @Summary List webhooks
// @Description List all webhooks
// @Tags webhooks
// @Produce json
// @Success 200 {array} Webhook
// @Failure 500 {object} map[string]interface{}
// @Router /api/webhooks [get]
func (ctrl *WebhookController) ListWebhooks(c *fiber.Ctx) error {
	webhooks, err := ctrl.Service.ListWebhooks(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": webhooks,
	})
}

// GetWebhook godoc
// GetWebhook godoc
// @Summary Get webhook
// @Description Get a webhook by ID
// @Tags webhooks
// @Produce json
// @Param id path string true "Webhook ID"
// @Success 200 {object} Webhook
// @Failure 404 {object} map[string]interface{}
// @Router /api/webhooks/{id} [get]
func (ctrl *WebhookController) GetWebhook(c *fiber.Ctx) error {
	id := c.Params("id")

	webhook, err := ctrl.Service.GetWebhook(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(webhook)
}

// UpdateWebhook godoc
// UpdateWebhook godoc
// @Summary Update webhook
// @Description Update an existing webhook
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook ID"
// @Param updates body map[string]interface{} true "Webhook Updates"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/webhooks/{id} [put]
func (ctrl *WebhookController) UpdateWebhook(c *fiber.Ctx) error {
	id := c.Params("id")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.Service.UpdateWebhook(c.UserContext(), id, updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Webhook updated successfully",
	})
}

// DeleteWebhook godoc
// DeleteWebhook godoc
// @Summary Delete webhook
// @Description Delete a webhook by ID
// @Tags webhooks
// @Param id path string true "Webhook ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/webhooks/{id} [delete]
func (ctrl *WebhookController) DeleteWebhook(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.Service.DeleteWebhook(c.UserContext(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Webhook deleted successfully",
	})
}
