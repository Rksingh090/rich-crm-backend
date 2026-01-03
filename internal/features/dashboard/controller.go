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
// CreateDashboard godoc
// @Summary Create dashboard
// @Description Create a new dashboard configuration
// @Tags dashboard
// @Accept json
// @Produce json
// @Param dashboard body DashboardConfig true "Dashboard Config"
// @Success 201 {object} DashboardConfig
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/dashboards [post]
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
// ListDashboards godoc
// @Summary List dashboards
// @Description List all dashboards for the current user
// @Tags dashboard
// @Produce json
// @Success 200 {array} DashboardConfig
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/dashboards [get]
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
// GetDashboard godoc
// @Summary Get dashboard
// @Description Get a dashboard configuration by ID
// @Tags dashboard
// @Produce json
// @Param id path string true "Dashboard ID"
// @Success 200 {object} DashboardConfig
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/dashboards/{id} [get]
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
// UpdateDashboard godoc
// @Summary Update dashboard
// @Description Update an existing dashboard configuration
// @Tags dashboard
// @Accept json
// @Produce json
// @Param id path string true "Dashboard ID"
// @Param dashboard body DashboardConfig true "Dashboard Config"
// @Success 200 {object} DashboardConfig
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/dashboards/{id} [put]
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
// DeleteDashboard godoc
// @Summary Delete dashboard
// @Description Delete a dashboard configuration
// @Tags dashboard
// @Param id path string true "Dashboard ID"
// @Success 204 {object} nil
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/dashboards/{id} [delete]
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
// SetDefaultDashboard godoc
// @Summary Set default dashboard
// @Description Set the default dashboard for the current user
// @Tags dashboard
// @Produce json
// @Param id path string true "Dashboard ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/dashboards/{id}/default [post]
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
// GetDashboardData godoc
// @Summary Get dashboard data
// @Description Get usage data for a dashboard
// @Tags dashboard
// @Produce json
// @Param id path string true "Dashboard ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/dashboards/{id}/data [get]
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
