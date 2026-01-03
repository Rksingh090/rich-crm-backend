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

// CreateRule godoc
// @Summary Create automation rule
// @Description Create a new automation rule
// @Tags automation
// @Accept json
// @Produce json
// @Param rule body AutomationRule true "Automation Rule"
// @Success 201 {object} AutomationRule
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/automation/rules [post]
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

// GetRule godoc
// @Summary Get automation rule
// @Description Get an automation rule by ID
// @Tags automation
// @Produce json
// @Param id path string true "Rule ID"
// @Success 200 {object} AutomationRule
// @Failure 404 {object} map[string]interface{}
// @Router /api/automation/rules/{id} [get]
func (ctrl *AutomationController) GetRule(c *fiber.Ctx) error {
	id := c.Params("id")
	rule, err := ctrl.Service.GetRule(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Rule not found"})
	}
	return c.JSON(rule)
}

// ListRules godoc
// @Summary List automation rules
// @Description List all automation rules, optionally filtered by module
// @Tags automation
// @Produce json
// @Param module_id query string false "Filter by Module ID"
// @Success 200 {array} AutomationRule
// @Failure 500 {object} map[string]interface{}
// @Router /api/automation/rules [get]
func (ctrl *AutomationController) ListRules(c *fiber.Ctx) error {
	moduleID := c.Query("module_id")
	rules, err := ctrl.Service.ListRules(c.UserContext(), moduleID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rules)
}

// UpdateRule godoc
// @Summary Update automation rule
// @Description Update an existing automation rule
// @Tags automation
// @Accept json
// @Produce json
// @Param id path string true "Rule ID"
// @Param rule body AutomationRule true "Automation Rule"
// @Success 200 {object} AutomationRule
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/automation/rules/{id} [put]
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

// DeleteRule godoc
// @Summary Delete automation rule
// @Description Delete an automation rule by ID
// @Tags automation
// @Param id path string true "Rule ID"
// @Success 204 {object} nil
// @Failure 500 {object} map[string]interface{}
// @Router /api/automation/rules/{id} [delete]
func (ctrl *AutomationController) DeleteRule(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.Service.DeleteRule(c.UserContext(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
