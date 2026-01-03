package email_template

import (
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EmailTemplateController struct {
	Service EmailTemplateService
}

func NewEmailTemplateController(service EmailTemplateService) *EmailTemplateController {
	return &EmailTemplateController{Service: service}
}

func (c *EmailTemplateController) Create(ctx *fiber.Ctx) error {
	var template EmailTemplate
	if err := ctx.BodyParser(&template); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := c.Service.CreateTemplate(ctx.UserContext(), &template); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(template)
}

func (c *EmailTemplateController) Get(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	template, err := c.Service.GetTemplate(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(template)
}

func (c *EmailTemplateController) List(ctx *fiber.Ctx) error {
	moduleName := ctx.Query("module")

	templates, err := c.Service.ListTemplates(ctx.UserContext(), moduleName, true)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(templates)
}

func (c *EmailTemplateController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	var template EmailTemplate
	if err := ctx.BodyParser(&template); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}
	template.ID = oid

	if err := c.Service.UpdateTemplate(ctx.UserContext(), &template); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(template)
}

func (c *EmailTemplateController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	if err := c.Service.DeleteTemplate(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (c *EmailTemplateController) GetModuleFields(ctx *fiber.Ctx) error {
	moduleName := ctx.Params("module")

	fields, err := c.Service.GetModuleFields(ctx.UserContext(), moduleName)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fields)
}

type TestEmailRequest struct {
	To       string                 `json:"to"`
	TestData map[string]interface{} `json:"test_data"`
}

// SendTestEmail sends a test email using the specified template
// @Summary Send a test email
// @Description Renders the email template with provided test data and sends it to the specified recipient
// @Tags EmailTemplates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Param request body TestEmailRequest true "Test Email Details"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/email-templates/{id}/test [post]
func (c *EmailTemplateController) SendTestEmail(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var req TestEmailRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if req.To == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "recipient email (to) is required"})
	}

	if err := c.Service.SendTestEmail(ctx.UserContext(), id, req.To, req.TestData); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Test email sent successfully"})
}
