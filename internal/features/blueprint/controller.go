package blueprint

import (
	"github.com/gofiber/fiber/v2"
)

type Controller struct {
	service Service
}

func NewController(service Service) *Controller {
	return &Controller{
		service: service,
	}
}

// CreateBlueprint godoc
// @Summary Create a new blueprint
// @Tags Blueprint
// @Accept json
// @Produce json
// @Param blueprint body Blueprint true "Blueprint"
// @Success 201 {object} Blueprint
// @Router /api/blueprints [post]
func (c *Controller) Create(ctx *fiber.Ctx) error {
	var blueprint Blueprint
	if err := ctx.BodyParser(&blueprint); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := c.service.Create(ctx.UserContext(), &blueprint); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(blueprint)
}

// UpdateBlueprint godoc
// @Summary Update an existing blueprint
// @Tags Blueprint
// @Accept json
// @Produce json
// @Param id path string true "Blueprint ID"
// @Param blueprint body Blueprint true "Blueprint"
// @Success 200 {object} Blueprint
// @Router /api/blueprints/{id} [put]
func (c *Controller) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var blueprint Blueprint
	if err := ctx.BodyParser(&blueprint); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := c.service.Update(ctx.UserContext(), id, &blueprint); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(blueprint)
}

// GetBlueprint godoc
// @Summary Get a blueprint by ID
// @Tags Blueprint
// @Produce json
// @Param id path string true "Blueprint ID"
// @Success 200 {object} Blueprint
// @Router /api/blueprints/{id} [get]
func (c *Controller) Get(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	blueprint, err := c.service.GetByID(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Blueprint not found"})
	}
	return ctx.JSON(blueprint)
}

// ListBlueprints godoc
// @Summary List blueprints by module
// @Tags Blueprint
// @Produce json
// @Param module query string false "Module Name"
// @Param search query string false "Search Term"
// @Success 200 {array} Blueprint
// @Router /api/blueprints [get]
func (c *Controller) List(ctx *fiber.Ctx) error {
	filter := BlueprintFilter{
		Module: ctx.Query("module"),
		Search: ctx.Query("search"),
	}

	blueprints, err := c.service.List(ctx.UserContext(), filter)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(blueprints)
}

// DeleteBlueprint godoc
// @Summary Delete a blueprint
// @Tags Blueprint
// @Param id path string true "Blueprint ID"
// @Success 204
// @Router /api/blueprints/{id} [delete]
func (c *Controller) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.service.Delete(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

type ExecuteTransitionRequest struct {
	TransitionID string         `json:"transition_id"`
	Data         map[string]any `json:"data"`
}

// ExecuteTransition godoc
// @Summary Execute a transition on a record
// @Tags Blueprint
// @Accept json
// @Produce json
// @Param module path string true "Module Name"
// @Param id path string true "Record ID"
// @Param request body ExecuteTransitionRequest true "Transition Request"
// @Success 200 {object} map[string]any
// @Router /api/blueprints/execute/{module}/{id} [post]
func (c *Controller) ExecuteTransition(ctx *fiber.Ctx) error {
	moduleName := ctx.Params("module")
	recordID := ctx.Params("id")

	var req ExecuteTransitionRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// We need to find the Active Blueprint for this module first
	blueprint, err := c.service.GetActiveByModule(ctx.UserContext(), moduleName)
	if err != nil || blueprint == nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No active blueprint found for this module"})
	}

	if err := c.service.ExecuteTransition(ctx.UserContext(), blueprint.ID.Hex(), req.TransitionID, recordID, req.Data); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"status": "success"})
}

// GetTransitions godoc
// @Summary Get available transitions for a record
// @Tags Blueprint
// @Produce json
// @Param module path string true "Module Name"
// @Param id path string true "Record ID"
// @Success 200 {array} Transition
// @Router /api/blueprints/transitions/{module}/{id} [get]
func (c *Controller) GetTransitions(ctx *fiber.Ctx) error {
	moduleName := ctx.Params("module")
	recordID := ctx.Params("id")

	transitions, err := c.service.GetAvailableTransitions(ctx.UserContext(), moduleName, recordID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(transitions)
}
