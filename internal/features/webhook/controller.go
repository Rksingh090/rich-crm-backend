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
