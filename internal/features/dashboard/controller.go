package dashboard

import (
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DashboardController struct {
	DashboardService DashboardService
}

func NewDashboardController(dashboardService DashboardService) *DashboardController {
	return &DashboardController{
		DashboardService: dashboardService,
	}
}

// CreateDashboard godoc
func (ctrl *DashboardController) CreateDashboard(ctx *fiber.Ctx) error {
	var dashboard DashboardConfig
	if err := ctx.BodyParser(&dashboard); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userIDStr := ctx.Locals("user_id")
	if userIDStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	if err := ctrl.DashboardService.CreateDashboard(ctx.UserContext(), &dashboard, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(dashboard)
}

// ListDashboards godoc
func (ctrl *DashboardController) ListDashboards(ctx *fiber.Ctx) error {
	userIDStr := ctx.Locals("user_id")
	if userIDStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	dashboards, err := ctrl.DashboardService.ListUserDashboards(ctx.UserContext(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(dashboards)
}

// GetDashboard godoc
func (ctrl *DashboardController) GetDashboard(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	userIDStr := ctx.Locals("user_id")
	if userIDStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	dashboard, err := ctrl.DashboardService.GetDashboard(ctx.UserContext(), id, userID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(dashboard)
}

// UpdateDashboard godoc
func (ctrl *DashboardController) UpdateDashboard(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	var dashboard DashboardConfig
	if err := ctx.BodyParser(&dashboard); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userIDStr := ctx.Locals("user_id")
	if userIDStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	if err := ctrl.DashboardService.UpdateDashboard(ctx.UserContext(), id, &dashboard, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(dashboard)
}

// DeleteDashboard godoc
func (ctrl *DashboardController) DeleteDashboard(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	userIDStr := ctx.Locals("user_id")
	if userIDStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	if err := ctrl.DashboardService.DeleteDashboard(ctx.UserContext(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

// SetDefaultDashboard godoc
func (ctrl *DashboardController) SetDefaultDashboard(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	userIDStr := ctx.Locals("user_id")
	if userIDStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	if err := ctrl.DashboardService.SetDefaultDashboard(ctx.UserContext(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Default dashboard set successfully"})
}

// GetDashboardData godoc
func (ctrl *DashboardController) GetDashboardData(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	userIDStr := ctx.Locals("user_id")
	if userIDStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	data, err := ctrl.DashboardService.GetDashboardData(ctx.UserContext(), id, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(data)
}
