package controllers

import (
	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DashboardController struct {
	DashboardService service.DashboardService
}

func NewDashboardController(dashboardService service.DashboardService) *DashboardController {
	return &DashboardController{
		DashboardService: dashboardService,
	}
}

// CreateDashboard creates a new dashboard
// @Summary Create dashboard
// @Description Create a new custom dashboard configuration
// @Tags dashboards
// @Accept json
// @Produce json
// @Param dashboard body models.DashboardConfig true "Dashboard configuration"
// @Success 201 {object} models.DashboardConfig
// @Router /api/dashboards [post]
func (ctrl *DashboardController) CreateDashboard(ctx *fiber.Ctx) error {
	var dashboard models.DashboardConfig
	if err := ctx.BodyParser(&dashboard); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Get user ID from context (set by auth middleware)
	userIDStr := ctx.Locals("userID")
	if userIDStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	if err := ctrl.DashboardService.CreateDashboard(ctx.Context(), &dashboard, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(dashboard)
}

// ListDashboards lists all dashboards for the current user
// @Summary List dashboards
// @Description Get all dashboards for the current user
// @Tags dashboards
// @Produce json
// @Success 200 {array} models.DashboardConfig
// @Router /api/dashboards [get]
func (ctrl *DashboardController) ListDashboards(ctx *fiber.Ctx) error {
	userIDStr := ctx.Locals("userID")
	if userIDStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	dashboards, err := ctrl.DashboardService.ListUserDashboards(ctx.Context(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(dashboards)
}

// GetDashboard gets a specific dashboard by ID
// @Summary Get dashboard
// @Description Get dashboard configuration by ID
// @Tags dashboards
// @Produce json
// @Param id path string true "Dashboard ID"
// @Success 200 {object} models.DashboardConfig
// @Router /api/dashboards/{id} [get]
func (ctrl *DashboardController) GetDashboard(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	userIDStr := ctx.Locals("userID")
	if userIDStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	dashboard, err := ctrl.DashboardService.GetDashboard(ctx.Context(), id, userID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(dashboard)
}

// UpdateDashboard updates a dashboard
// @Summary Update dashboard
// @Description Update dashboard configuration
// @Tags dashboards
// @Accept json
// @Produce json
// @Param id path string true "Dashboard ID"
// @Param dashboard body models.DashboardConfig true "Dashboard configuration"
// @Success 200 {object} models.DashboardConfig
// @Router /api/dashboards/{id} [put]
func (ctrl *DashboardController) UpdateDashboard(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	var dashboard models.DashboardConfig
	if err := ctx.BodyParser(&dashboard); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userIDStr := ctx.Locals("userID")
	if userIDStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	if err := ctrl.DashboardService.UpdateDashboard(ctx.Context(), id, &dashboard, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(dashboard)
}

// DeleteDashboard deletes a dashboard
// @Summary Delete dashboard
// @Description Delete a dashboard by ID
// @Tags dashboards
// @Param id path string true "Dashboard ID"
// @Success 204
// @Router /api/dashboards/{id} [delete]
func (ctrl *DashboardController) DeleteDashboard(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	userIDStr := ctx.Locals("userID")
	if userIDStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	if err := ctrl.DashboardService.DeleteDashboard(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

// SetDefaultDashboard sets a dashboard as the default for the user
// @Summary Set default dashboard
// @Description Set a dashboard as the default dashboard for the current user
// @Tags dashboards
// @Param id path string true "Dashboard ID"
// @Success 200
// @Router /api/dashboards/{id}/set-default [post]
func (ctrl *DashboardController) SetDefaultDashboard(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	userIDStr := ctx.Locals("userID")
	if userIDStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	if err := ctrl.DashboardService.SetDefaultDashboard(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Default dashboard set successfully"})
}

// GetDashboardData gets the data for all widgets in a dashboard
// @Summary Get dashboard data
// @Description Fetch data for all widgets in a dashboard
// @Tags dashboards
// @Produce json
// @Param id path string true "Dashboard ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/dashboards/{id}/data [get]
func (ctrl *DashboardController) GetDashboardData(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	userIDStr := ctx.Locals("userID")
	if userIDStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	data, err := ctrl.DashboardService.GetDashboardData(ctx.Context(), id, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(data)
}
