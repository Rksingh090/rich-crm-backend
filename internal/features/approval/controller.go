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

// CreateWorkflow godoc
// @Summary Create a new approval workflow
// @Description Create a new approval workflow configuration
// @Tags approvals
// @Accept json
// @Produce json
// @Param workflow body ApprovalWorkflow true "Workflow Configuration"
// @Success 201 {object} map[string]string "Workflow created successfully"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/approvals/workflows [post]
func (c *ApprovalController) CreateWorkflow(ctx *fiber.Ctx) error {
	var input ApprovalWorkflow
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.Service.CreateWorkflow(ctx.UserContext(), input); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Workflow created successfully"})
}

// UpdateWorkflow godoc
// @Summary Update an approval workflow
// @Description Update an existing approval workflow configuration
// @Tags approvals
// @Accept json
// @Produce json
// @Param id path string true "Workflow ID"
// @Param workflow body ApprovalWorkflow true "Workflow Configuration"
// @Success 200 {object} map[string]string "Workflow updated successfully"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/approvals/workflows/{id} [put]
func (c *ApprovalController) UpdateWorkflow(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var input ApprovalWorkflow
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.Service.UpdateWorkflow(ctx.UserContext(), id, input); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Workflow updated successfully"})
}

// DeleteWorkflow godoc
// @Summary Delete an approval workflow
// @Description Delete an approval workflow configuration
// @Tags approvals
// @Param id path string true "Workflow ID"
// @Success 204 {object} nil "No Content"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/approvals/workflows/{id} [delete]
func (c *ApprovalController) DeleteWorkflow(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.Service.DeleteWorkflow(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// GetWorkflowByModule godoc
// @Summary Get workflow by module
// @Description Get the active approval workflow for a specific module
// @Tags approvals
// @Produce json
// @Param moduleId path string true "Module ID"
// @Success 200 {object} ApprovalWorkflow
// @Failure 404 {object} map[string]string "No active workflow found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/approvals/workflows/module/{moduleId} [get]
func (c *ApprovalController) GetWorkflowByModule(ctx *fiber.Ctx) error {
	moduleID := ctx.Params("moduleId")
	workflow, err := c.Service.GetWorkflowByModule(ctx.UserContext(), moduleID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if workflow == nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No active workflow found for this module"})
	}
	return ctx.JSON(workflow)
}

// GetWorkflowByID godoc
// @Summary Get workflow by ID
// @Description Get a specific approval workflow by its ID
// @Tags approvals
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {object} ApprovalWorkflow
// @Failure 404 {object} map[string]string "Workflow not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/approvals/workflows/{id} [get]
func (c *ApprovalController) GetWorkflowByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	workflow, err := c.Service.GetWorkflowByID(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if workflow == nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Workflow not found"})
	}
	return ctx.JSON(workflow)
}

// ListWorkflows godoc
// @Summary List all workflows
// @Description List all approval workflows
// @Tags approvals
// @Produce json
// @Success 200 {array} ApprovalWorkflow
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/approvals/workflows [get]
func (c *ApprovalController) ListWorkflows(ctx *fiber.Ctx) error {
	workflows, err := c.Service.ListWorkflows(ctx.UserContext())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(workflows)
}

// ApproveRecord godoc
// @Summary Approve a record
// @Description Approve a record for the current step in the workflow
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
// @Description Reject a record for the current step in the workflow
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
