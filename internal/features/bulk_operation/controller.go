package bulk_operation

import (
	"context"

	common_models "go-crm/internal/common/models"

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

// PreviewBulkOperation godoc
// @Summary Preview bulk operation
// @Description Preview the effect of a bulk operation
// @Tags bulk_operations
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "Preview Request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/bulk/preview [post]
func (c *BulkOperationController) PreviewBulkOperation(ctx *fiber.Ctx) error {
	var req struct {
		ModuleName string                 `json:"module_name"`
		Filters    []common_models.Filter `json:"filters"`
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

// CreateBulkOperation godoc
// @Summary Create bulk operation
// @Description Create a new bulk operation job
// @Tags bulk_operations
// @Accept json
// @Produce json
// @Param operation body BulkOperation true "Bulk Operation"
// @Success 201 {object} BulkOperation
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/bulk/operations [post]
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

	tenantIDStr, ok := ctx.Locals("tenant_id").(string)
	if ok && tenantIDStr != "" {
		tenantID, _ := primitive.ObjectIDFromHex(tenantIDStr)
		op.TenantID = tenantID
	}

	if op.Type == "" {
		op.Type = BulkTypeUpdate
	}
	if op.Type != BulkTypeUpdate && op.Type != BulkTypeDelete && op.Type != BulkTypeDuplicate {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid operation type"})
	}

	if err := c.BulkService.CreateBulkOperation(ctx.UserContext(), &op); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(op)
}

// ExecuteBulkOperation godoc
// @Summary Execute bulk operation
// @Description Trigger execution of a bulk operation
// @Tags bulk_operations
// @Produce json
// @Param id path string true "Operation ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/bulk/operations/{id}/execute [post]
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

// GetBulkOperation godoc
// @Summary Get bulk operation
// @Description Get details of a bulk operation
// @Tags bulk_operations
// @Produce json
// @Param id path string true "Operation ID"
// @Success 200 {object} BulkOperation
// @Failure 404 {object} map[string]interface{}
// @Router /api/bulk/operations/{id} [get]
func (c *BulkOperationController) GetBulkOperation(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	op, err := c.BulkService.GetOperation(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Operation not found"})
	}

	return ctx.JSON(op)
}

// ListBulkOperations godoc
// @Summary List bulk operations
// @Description List all bulk operations for the current user
// @Tags bulk_operations
// @Produce json
// @Success 200 {array} BulkOperation
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/bulk/operations [get]
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
