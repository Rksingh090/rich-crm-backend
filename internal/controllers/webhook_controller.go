package controllers

import (
	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WebhookController struct {
	Service service.WebhookService
}

func NewWebhookController(service service.WebhookService) *WebhookController {
	return &WebhookController{
		Service: service,
	}
}

// CreateWebhook godoc
// @Summary      Create a new webhook
// @Description  Register a new webhook url
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Param        input body models.Webhook true "Webhook Data"
// @Success      201  {object} map[string]interface{}
// @Failure      400  {string} string "Invalid request"
// @Router       /api/webhooks [post]
func (ctrl *WebhookController) CreateWebhook(c *fiber.Ctx) error {
	var webhook models.Webhook
	if err := c.BodyParser(&webhook); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userIDStr, ok := c.Locals("userID").(string)
	if ok {
		if oid, err := primitive.ObjectIDFromHex(userIDStr); err == nil {
			webhook.CreatedBy = oid
		}
	}

	if err := ctrl.Service.CreateWebhook(c.Context(), &webhook); err != nil {
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
// @Summary      List all webhooks
// @Description  Get a list of all webhooks
// @Tags         webhooks
// @Produce      json
// @Success      200 {array} models.Webhook
// @Router       /api/webhooks [get]
func (ctrl *WebhookController) ListWebhooks(c *fiber.Ctx) error {
	webhooks, err := ctrl.Service.ListWebhooks(c.Context())
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
// @Summary      Get a webhook
// @Description  Get a webhook by ID
// @Tags         webhooks
// @Produce      json
// @Param        id path string true "Webhook ID"
// @Success      200 {object} models.Webhook
// @Failure      404 {string} string "Not Found"
// @Router       /api/webhooks/{id} [get]
func (ctrl *WebhookController) GetWebhook(c *fiber.Ctx) error {
	id := c.Params("id")

	webhook, err := ctrl.Service.GetWebhook(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(webhook)
}

// UpdateWebhook godoc
// @Summary      Update a webhook
// @Description  Update a webhook by ID
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Param        id    path string true "Webhook ID"
// @Param        input body map[string]interface{} true "Update Data"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid input"
// @Router       /api/webhooks/{id} [put]
func (ctrl *WebhookController) UpdateWebhook(c *fiber.Ctx) error {
	id := c.Params("id")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.Service.UpdateWebhook(c.Context(), id, updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Webhook updated successfully",
	})
}

// DeleteWebhook godoc
// @Summary      Delete a webhook
// @Description  Delete a webhook by ID
// @Tags         webhooks
// @Produce      json
// @Param        id path string true "Webhook ID"
// @Success      200 {object} map[string]string
// @Failure      400 {string} string "Invalid input"
// @Router       /api/webhooks/{id} [delete]
func (ctrl *WebhookController) DeleteWebhook(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.Service.DeleteWebhook(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Webhook deleted successfully",
	})
}
