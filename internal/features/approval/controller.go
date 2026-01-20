package approval

import (
	"go-crm/internal/core/auth"
	"go-crm/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ApprovalController struct {
	Service     ApprovalService
	AuthService auth.AuthService
}

func NewApprovalController(service ApprovalService, authService auth.AuthService) *ApprovalController {
	return &ApprovalController{
		Service:     service,
		AuthService: authService,
	}
}

// CreateApprovalProcess godoc
// @Summary Create a new approval process
// @Description Create a new approval process configuration
// @Tags approvals
// @Accept json
// @Produce json
// @Param process body ApprovalProcess true "Approval Process Configuration"
// @Success 201 {object} map[string]string "Approval process created successfully"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/approval-processes [post]
func (c *ApprovalController) CreateApprovalProcess(ctx *fiber.Ctx) error {
	var input ApprovalProcess
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var userID primitive.ObjectID
	if idStr, ok := ctx.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	createdProcess, err := c.Service.CreateApprovalProcess(ctx.UserContext(), input, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(createdProcess)
}

// UpdateApprovalProcess godoc
// @Summary Update an approval process
// @Description Update an existing approval process configuration
// @Tags approvals
// @Accept json
// @Produce json
// @Param id path string true "Approval Process ID"
// @Param process body ApprovalProcess true "Approval Process Configuration"
// @Success 200 {object} map[string]string "Approval process updated successfully"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/approval-processes/{id} [put]
func (c *ApprovalController) UpdateApprovalProcess(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var input ApprovalProcess
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var userID primitive.ObjectID
	if idStr, ok := ctx.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	if err := c.Service.UpdateApprovalProcess(ctx.UserContext(), id, input, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Approval process updated successfully"})
}

// DeleteApprovalProcess godoc
// @Summary Delete an approval process
// @Description Delete an approval process configuration
// @Tags approvals
// @Param id path string true "Approval Process ID"
// @Success 204 {object} nil "No Content"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/approval-processes/{id} [delete]
func (c *ApprovalController) DeleteApprovalProcess(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var userID primitive.ObjectID
	if idStr, ok := ctx.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	if err := c.Service.DeleteApprovalProcess(ctx.UserContext(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// GetApprovalProcessByModule godoc
// @Summary Get approval process by module
// @Description Get the active approval process for a specific module
// @Tags approvals
// @Produce json
// @Param moduleId path string true "Module ID"
// @Success 200 {object} ApprovalProcess
// @Failure 404 {object} map[string]string "No active process found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/approval-processes/module/{moduleId} [get]
func (c *ApprovalController) GetApprovalProcessByModule(ctx *fiber.Ctx) error {
	moduleID := ctx.Params("moduleId")
	var userID primitive.ObjectID
	if idStr, ok := ctx.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	process, err := c.Service.GetApprovalProcessByModule(ctx.UserContext(), moduleID, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if process == nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No active approval process found for this module"})
	}
	return ctx.JSON(process)
}

// GetApprovalProcessByID godoc
// @Summary Get approval process by ID
// @Description Get a specific approval process by its ID
// @Tags approvals
// @Produce json
// @Param id path string true "Approval Process ID"
// @Success 200 {object} ApprovalProcess
// @Failure 404 {object} map[string]string "Approval process not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/approval-processes/{id} [get]
func (c *ApprovalController) GetApprovalProcessByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var userID primitive.ObjectID
	if idStr, ok := ctx.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	process, err := c.Service.GetApprovalProcessByID(ctx.UserContext(), id, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if process == nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Approval process not found"})
	}
	return ctx.JSON(process)
}

// ListApprovalProcesses godoc
// @Summary List all approval processes
// @Description List all approval processes
// @Tags approvals
// @Produce json
// @Success 200 {array} ApprovalProcess
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/approval-processes [get]
func (c *ApprovalController) ListApprovalProcesses(ctx *fiber.Ctx) error {
	var userID primitive.ObjectID
	if idStr, ok := ctx.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	processes, err := c.Service.ListApprovalProcesses(ctx.UserContext(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(processes)
}

// ApproveRecord godoc
// @Summary Approve a record
// @Description Approve a record for the current step in the approval process
// @Tags approvals
// @Accept json
// @Produce json
// @Param module path string true "Module Name"
// @Param id path string true "Record ID"
// @Param body body map[string]string true "Approval Comment"
// @Success 200 {object} map[string]string "Record approved successfully"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/approvals/{module}/{id}/approve [post]
func (c *ApprovalController) ApproveRecord(ctx *fiber.Ctx) error {
	moduleName := ctx.Params("module")
	recordID := ctx.Params("id")

	var body struct {
		Comment string `json:"comment"`
	}
	_ = ctx.BodyParser(&body)

	userClaims := ctx.Locals(utils.UserClaimsKey).(*utils.UserClaims)

	canApprove, err := c.Service.CanApprove(ctx.UserContext(), moduleName, recordID, userClaims.UserID, userClaims.RoleIDs)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if !canApprove {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You are not authorized to approve this step"})
	}

	if err := c.Service.ApproveRecord(ctx.UserContext(), moduleName, recordID, userClaims.UserID, body.Comment); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Record approved successfully"})
}

// RejectRecord godoc
// @Summary Reject a record
// @Description Reject a record for the current step in the approval process
// @Tags approvals
// @Accept json
// @Produce json
// @Param module path string true "Module Name"
// @Param id path string true "Record ID"
// @Param body body map[string]string true "Rejection Comment"
// @Success 200 {object} map[string]string "Record rejected successfully"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/approvals/{module}/{id}/reject [post]
func (c *ApprovalController) RejectRecord(ctx *fiber.Ctx) error {
	moduleName := ctx.Params("module")
	recordID := ctx.Params("id")

	var body struct {
		Comment string `json:"comment"`
	}
	_ = ctx.BodyParser(&body)

	userClaims := ctx.Locals(utils.UserClaimsKey).(*utils.UserClaims)

	canApprove, err := c.Service.CanApprove(ctx.UserContext(), moduleName, recordID, userClaims.UserID, userClaims.RoleIDs)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if !canApprove {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You are not authorized to reject this step"})
	}

	if err := c.Service.RejectRecord(ctx.UserContext(), moduleName, recordID, userClaims.UserID, body.Comment); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Record rejected successfully"})
}
