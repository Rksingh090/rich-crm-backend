package bulk_operation

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BulkOperationController struct {
	BulkService BulkOperationService
}

func NewBulkOperationController(bulkService BulkOperationService) *BulkOperationController {
	return &BulkOperationController{
		BulkService: bulkService,
	}
}

func (c *BulkOperationController) PreviewBulkOperation(ctx *fiber.Ctx) error {
	var req struct {
		ModuleName string                 `json:"module_name"`
		Filters    map[string]interface{} `json:"filters"`
	}

	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	userIDStr, ok := ctx.Locals("user_id").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	records, total, err := c.BulkService.PreviewBulkOperation(ctx.UserContext(), req.ModuleName, req.Filters, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{
		"records": records,
		"total":   total,
	})
}

func (c *BulkOperationController) CreateBulkOperation(ctx *fiber.Ctx) error {
	var op BulkOperation
	if err := ctx.BodyParser(&op); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	userIDStr, ok := ctx.Locals("user_id").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)
	op.UserID = userID

	if op.Type == "" {
		op.Type = BulkTypeUpdate
	}
	if op.Type != BulkTypeUpdate && op.Type != BulkTypeDelete {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid operation type"})
	}

	if err := c.BulkService.CreateBulkOperation(ctx.UserContext(), &op); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(op)
}

func (c *BulkOperationController) ExecuteBulkOperation(ctx *fiber.Ctx) error {
	opID := ctx.Params("id")

	userIDStr, ok := ctx.Locals("user_id").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	go func() {
		bgCtx := context.Background()
		c.BulkService.ExecuteBulkOperation(bgCtx, opID, userID)
	}()

	return ctx.JSON(fiber.Map{"message": "Bulk operation started"})
}

func (c *BulkOperationController) GetBulkOperation(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	op, err := c.BulkService.GetOperation(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Operation not found"})
	}

	return ctx.JSON(op)
}

func (c *BulkOperationController) ListBulkOperations(ctx *fiber.Ctx) error {
	userIDStr, ok := ctx.Locals("user_id").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	ops, err := c.BulkService.GetUserOperations(ctx.UserContext(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(ops)
}
