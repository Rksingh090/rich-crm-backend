package approval

import (
	"go-crm/internal/features/auth"
	"go-crm/pkg/utils"

	"github.com/gofiber/fiber/v2"
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

func (c *ApprovalController) CreateWorkflow(ctx *fiber.Ctx) error {
	var input ApprovalWorkflow
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.Service.CreateWorkflow(ctx.Context(), input); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Workflow created successfully"})
}

func (c *ApprovalController) UpdateWorkflow(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var input ApprovalWorkflow
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.Service.UpdateWorkflow(ctx.Context(), id, input); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Workflow updated successfully"})
}

func (c *ApprovalController) DeleteWorkflow(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.Service.DeleteWorkflow(ctx.Context(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (c *ApprovalController) GetWorkflowByModule(ctx *fiber.Ctx) error {
	moduleID := ctx.Params("moduleId")
	workflow, err := c.Service.GetWorkflowByModule(ctx.Context(), moduleID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if workflow == nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No active workflow found for this module"})
	}
	return ctx.JSON(workflow)
}

func (c *ApprovalController) GetWorkflowByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	workflow, err := c.Service.GetWorkflowByID(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if workflow == nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Workflow not found"})
	}
	return ctx.JSON(workflow)
}

func (c *ApprovalController) ListWorkflows(ctx *fiber.Ctx) error {
	workflows, err := c.Service.ListWorkflows(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(workflows)
}

func (c *ApprovalController) ApproveRecord(ctx *fiber.Ctx) error {
	moduleName := ctx.Params("module")
	recordID := ctx.Params("id")

	var body struct {
		Comment string `json:"comment"`
	}
	_ = ctx.BodyParser(&body)

	userClaims := ctx.Locals(utils.UserClaimsKey).(*utils.UserClaims)

	canApprove, err := c.Service.CanApprove(ctx.Context(), moduleName, recordID, userClaims.UserID, userClaims.RoleIDs)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if !canApprove {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You are not authorized to approve this step"})
	}

	if err := c.Service.ApproveRecord(ctx.Context(), moduleName, recordID, userClaims.UserID, body.Comment); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Record approved successfully"})
}

func (c *ApprovalController) RejectRecord(ctx *fiber.Ctx) error {
	moduleName := ctx.Params("module")
	recordID := ctx.Params("id")

	var body struct {
		Comment string `json:"comment"`
	}
	_ = ctx.BodyParser(&body)

	userClaims := ctx.Locals(utils.UserClaimsKey).(*utils.UserClaims)

	canApprove, err := c.Service.CanApprove(ctx.Context(), moduleName, recordID, userClaims.UserID, userClaims.RoleIDs)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if !canApprove {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You are not authorized to reject this step"})
	}

	if err := c.Service.RejectRecord(ctx.Context(), moduleName, recordID, userClaims.UserID, body.Comment); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Record rejected successfully"})
}
