package function

import (
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FunctionController struct {
	Service FunctionService
}

func NewFunctionController(service FunctionService) *FunctionController {
	return &FunctionController{Service: service}
}

// Create godoc
// @Summary Create function
// @Description Create a new reusable function
// @Tags functions
// @Accept json
// @Produce json
// @Param function body Function true "Function"
// @Success 201 {object} Function
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/functions [post]
func (c *FunctionController) Create(ctx *fiber.Ctx) error {
	var function Function
	if err := ctx.BodyParser(&function); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Set App from header
	app := ctx.Get("X-Rich-App", "crm")
	function.App = app

	if err := c.Service.CreateFunction(ctx.UserContext(), &function); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(function)
}

// Get godoc
// @Summary Get function
// @Description Get a function by ID
// @Tags functions
// @Produce json
// @Param id path string true "Function ID"
// @Success 200 {object} Function
// @Failure 404 {object} map[string]any
// @Router /api/functions/{id} [get]
func (c *FunctionController) Get(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	function, err := c.Service.GetFunction(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(function)
}

// List godoc
// @Summary List functions
// @Description List all functions, optionally filtered by module
// @Tags functions
// @Produce json
// @Param module query string false "Filter by module"
// @Success 200 {array} Function
// @Failure 500 {object} map[string]any
// @Router /api/functions [get]
func (c *FunctionController) List(ctx *fiber.Ctx) error {
	moduleName := ctx.Query("module")

	functions, err := c.Service.ListFunctions(ctx.UserContext(), moduleName, true)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(functions)
}

// Update godoc
// @Summary Update function
// @Description Update an existing function
// @Tags functions
// @Accept json
// @Produce json
// @Param id path string true "Function ID"
// @Param function body Function true "Function"
// @Success 200 {object} Function
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/functions/{id} [put]
func (c *FunctionController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	var function Function
	if err := ctx.BodyParser(&function); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}
	function.ID = oid

	// Set App from header
	app := ctx.Get("X-Rich-App", "crm")
	function.App = app

	if err := c.Service.UpdateFunction(ctx.UserContext(), &function); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(function)
}

// Delete godoc
// @Summary Delete function
// @Description Delete a function by ID
// @Tags functions
// @Param id path string true "Function ID"
// @Success 204 {object} nil
// @Failure 500 {object} map[string]any
// @Router /api/functions/{id} [delete]
func (c *FunctionController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	if err := c.Service.DeleteFunction(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

type TestFunctionRequest struct {
	TestData map[string]any `json:"test_data"`
	Code     string         `json:"code"`
}

// TestFunction godoc
// @Summary Test a function
// @Description Execute a function with test data and return the result
// @Tags functions
// @Accept json
// @Produce json
// @Param id path string true "Function ID"
// @Param request body TestFunctionRequest true "Test Data"
// @Success 200 {object} map[string]any
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/functions/{id}/test [post]
func (c *FunctionController) TestFunction(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var req TestFunctionRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	result, err := c.Service.TestFunction(ctx.UserContext(), id, req.TestData, req.Code)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"result":  result,
	})
}
