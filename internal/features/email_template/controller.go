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

	if err := c.Service.CreateTemplate(ctx.Context(), &template); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(template)
}

func (c *EmailTemplateController) Get(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	template, err := c.Service.GetTemplate(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(template)
}

func (c *EmailTemplateController) List(ctx *fiber.Ctx) error {
	moduleName := ctx.Query("module")

	templates, err := c.Service.ListTemplates(ctx.Context(), moduleName, true)
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

	if err := c.Service.UpdateTemplate(ctx.Context(), &template); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(template)
}

func (c *EmailTemplateController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	if err := c.Service.DeleteTemplate(ctx.Context(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (c *EmailTemplateController) GetModuleFields(ctx *fiber.Ctx) error {
	moduleName := ctx.Params("module")

	fields, err := c.Service.GetModuleFields(ctx.Context(), moduleName)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fields)
}
