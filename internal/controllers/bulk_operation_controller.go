package controllers

import (
	"context"
	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BulkOperationController struct {
	BulkService service.BulkOperationService
}

func NewBulkOperationController(bulkService service.BulkOperationService) *BulkOperationController {
	return &BulkOperationController{
		BulkService: bulkService,
	}
}

// PreviewBulkOperation godoc
// @Summary Preview bulk operation
// @Tags Bulk Operations
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "Preview request"
// @Success 200 {object} map[string]interface{}
// @Router /api/bulk/preview [post]
func (c *BulkOperationController) PreviewBulkOperation(ctx *fiber.Ctx) error {
	var req struct {
		ModuleName string                 `json:"module_name"`
		Filters    map[string]interface{} `json:"filters"`
	}

	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	userIDStr, ok := ctx.Locals("userID").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	records, total, err := c.BulkService.PreviewBulkOperation(ctx.Context(), req.ModuleName, req.Filters, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{
		"records": records,
		"total":   total,
	})
}

// CreateBulkOperation godoc
// @Summary Create bulk operation
// @Tags Bulk Operations
// @Accept json
// @Produce json
// @Param operation body models.BulkOperation true "Bulk Operation"
// @Success 201 {object} models.BulkOperation
// @Router /api/bulk/operations [post]
func (c *BulkOperationController) CreateBulkOperation(ctx *fiber.Ctx) error {
	var op models.BulkOperation
	if err := ctx.BodyParser(&op); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	userIDStr, ok := ctx.Locals("userID").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)
	op.UserID = userID

	// Validate Type
	if op.Type == "" {
		op.Type = models.BulkTypeUpdate // Default
	}
	if op.Type != models.BulkTypeUpdate && op.Type != models.BulkTypeDelete {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid operation type"})
	}

	if err := c.BulkService.CreateBulkOperation(ctx.Context(), &op); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(op)
}

// ExecuteBulkOperation godoc
// @Summary Execute bulk operation
// @Tags Bulk Operations
// @Param id path string true "Operation ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/bulk/operations/{id}/execute [post]
func (c *BulkOperationController) ExecuteBulkOperation(ctx *fiber.Ctx) error {
	opID := ctx.Params("id")

	userIDStr, ok := ctx.Locals("userID").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	// Execute in background
	go func() {
		bgCtx := context.Background()
		c.BulkService.ExecuteBulkOperation(bgCtx, opID, userID)
	}()

	return ctx.JSON(fiber.Map{"message": "Bulk operation started"})
}

// GetBulkOperation godoc
// @Summary Get bulk operation status
// @Tags Bulk Operations
// @Produce json
// @Param id path string true "Operation ID"
// @Success 200 {object} models.BulkOperation
// @Router /api/bulk/operations/{id} [get]
func (c *BulkOperationController) GetBulkOperation(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	op, err := c.BulkService.GetOperation(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Operation not found"})
	}

	return ctx.JSON(op)
}

// ListBulkOperations godoc
// @Summary List user's bulk operations
// @Tags Bulk Operations
// @Produce json
// @Success 200 {array} models.BulkOperation
// @Router /api/bulk/operations [get]
func (c *BulkOperationController) ListBulkOperations(ctx *fiber.Ctx) error {
	userIDStr, ok := ctx.Locals("userID").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	ops, err := c.BulkService.GetUserOperations(ctx.Context(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(ops)
}
