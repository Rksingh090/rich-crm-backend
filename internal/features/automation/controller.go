package automation

import (
	"github.com/gofiber/fiber/v2"
)

type AutomationController struct {
	Service AutomationService
}

func NewAutomationController(service AutomationService) *AutomationController {
	return &AutomationController{
		Service: service,
	}
}

func (ctrl *AutomationController) CreateRule(c *fiber.Ctx) error {
	var rule AutomationRule
	if err := c.BodyParser(&rule); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := ctrl.Service.CreateRule(c.UserContext(), &rule); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(rule)
}

func (ctrl *AutomationController) GetRule(c *fiber.Ctx) error {
	id := c.Params("id")
	rule, err := ctrl.Service.GetRule(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Rule not found"})
	}
	return c.JSON(rule)
}

func (ctrl *AutomationController) ListRules(c *fiber.Ctx) error {
	moduleID := c.Query("module_id")
	rules, err := ctrl.Service.ListRules(c.UserContext(), moduleID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rules)
}

func (ctrl *AutomationController) UpdateRule(c *fiber.Ctx) error {
	var rule AutomationRule
	if err := c.BodyParser(&rule); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Ensure ID is set from path
	// (Assuming ID is string or ObjectID)
	if err := ctrl.Service.UpdateRule(c.UserContext(), &rule); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(rule)
}

func (ctrl *AutomationController) DeleteRule(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.Service.DeleteRule(c.UserContext(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
