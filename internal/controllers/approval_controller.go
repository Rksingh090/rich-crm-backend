package controllers

import (
	"go-crm/internal/models"
	"go-crm/internal/service"
	"go-crm/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ApprovalController struct {
	Service     service.ApprovalService
	AuthService service.AuthService
}

func NewApprovalController(service service.ApprovalService, authService service.AuthService) *ApprovalController {
	return &ApprovalController{
		Service:     service,
		AuthService: authService,
	}
}

// CreateWorkflow godoc
// @Summary      Create approval workflow
// @Description  Create a new approval workflow for a module
// @Tags         approvals
// @Accept       json
// @Produce      json
// @Param        input body models.ApprovalWorkflow true "Workflow Input"
// @Success      201  {object} map[string]string
// @Failure      400  {string} string "Invalid request body"
// @Failure      500  {string} string "Failed to create workflow"
// @Router       /workflows [post]
func (c *ApprovalController) CreateWorkflow(ctx *fiber.Ctx) error {
	var input models.ApprovalWorkflow
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.Service.CreateWorkflow(ctx.Context(), input); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Workflow created successfully"})
}

// GetWorkflow godoc
// @Summary      Get workflow by module
// @Description  Get the active workflow for a specific module
// @Tags         approvals
// @Accept       json
// @Produce      json
// @Param        moduleId path string true "Module ID (hex)"
// @Success      200  {object} models.ApprovalWorkflow
// @Failure      404  {string} string "Workflow not found"
// @Router       /workflows/{moduleId} [get]
func (c *ApprovalController) GetWorkflow(ctx *fiber.Ctx) error {
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

// ListWorkflows godoc
// @Summary      List all workflows
// @Description  List all approval workflows
// @Tags         approvals
// @Accept       json
// @Produce      json
// @Success      200  {array} models.ApprovalWorkflow
// @Router       /workflows [get]
func (c *ApprovalController) ListWorkflows(ctx *fiber.Ctx) error {
	workflows, err := c.Service.ListWorkflows(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(workflows)
}

// ApproveRecord godoc
// @Summary      Approve a record
// @Description  Approve the current step for a record
// @Tags         approvals
// @Accept       json
// @Produce      json
// @Param        module path string true "Module Name"
// @Param        id path string true "Record ID"
// @Param        input body map[string]string true "Comment"
// @Success      200  {object} map[string]string
// @Failure      403  {string} string "Forbidden"
// @Router       /approvals/{module}/{id}/approve [post]
func (c *ApprovalController) ApproveRecord(ctx *fiber.Ctx) error {
	moduleName := ctx.Params("module")
	recordID := ctx.Params("id")

	var body struct {
		Comment string `json:"comment"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		// Ignore check, optional comment
	}

	userClaims := ctx.Locals(utils.UserClaimsKey).(*utils.UserClaims)

	// Check Permission
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

// RejectRecord godoc
// @Summary      Reject a record
// @Description  Reject the current step for a record
// @Tags         approvals
// @Accept       json
// @Produce      json
// @Param        module path string true "Module Name"
// @Param        id path string true "Record ID"
// @Param        input body map[string]string true "Comment"
// @Success      200  {object} map[string]string
// @Failure      403  {string} string "Forbidden"
// @Router       /approvals/{module}/{id}/reject [post]
func (c *ApprovalController) RejectRecord(ctx *fiber.Ctx) error {
	moduleName := ctx.Params("module")
	recordID := ctx.Params("id")

	var body struct {
		Comment string `json:"comment"`
	}
	ctx.BodyParser(&body)

	userClaims := ctx.Locals(utils.UserClaimsKey).(*utils.UserClaims)

	// Check Permission
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
