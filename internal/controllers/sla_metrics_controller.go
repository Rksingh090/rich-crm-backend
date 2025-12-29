package controllers

import (
	"time"

	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

// SLAMetricsController handles SLA metrics and tracking endpoints
type SLAMetricsController struct {
	slaService    service.SLAService
	ticketService service.TicketService
}

// NewSLAMetricsController creates a new SLA metrics controller
func NewSLAMetricsController(slaService service.SLAService, ticketService service.TicketService) *SLAMetricsController {
	return &SLAMetricsController{
		slaService:    slaService,
		ticketService: ticketService,
	}
}

// GetOverview retrieves overall SLA performance metrics
// @Summary Get SLA overview metrics
// @Description Retrieve aggregated SLA performance metrics including compliance rate, averages, etc.
// @Tags SLA Metrics
// @Accept json
// @Produce json
// @Param days query int false "Number of days to analyze" default(30)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/sla-metrics/overview [get]
func (c *SLAMetricsController) GetOverview(ctx *fiber.Ctx) error {
	// Get days parameter (default to 30)
	days := ctx.QueryInt("days", 30)

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	metrics, err := c.slaService.GetSLAMetrics(ctx.Context(), startDate, endDate)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to fetch SLA metrics",
			"details": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    metrics,
	})
}

// GetViolations retrieves all active SLA violations
// @Summary Get active SLA violations
// @Description Retrieve list of all tickets currently in SLA breach status
// @Tags SLA Metrics
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/sla-metrics/violations [get]
func (c *SLAMetricsController) GetViolations(ctx *fiber.Ctx) error {
	violations, err := c.slaService.GetSLAViolations(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to fetch SLA violations",
			"details": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    violations,
		"total":   len(violations),
	})
}

// GetTrends retrieves SLA performance trends over time
// @Summary Get SLA performance trends
// @Description Retrieve historical SLA performance data for trend analysis
// @Tags SLA Metrics
// @Accept json
// @Produce json
// @Param days query int false "Number of days to analyze" default(30)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/sla-metrics/trends [get]
func (c *SLAMetricsController) GetTrends(ctx *fiber.Ctx) error {
	days := ctx.QueryInt("days", 30)

	trends, err := c.slaService.GetSLATrends(ctx.Context(), days)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to fetch SLA trends",
			"details": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    trends,
	})
}

// GetTicketSLAStatus retrieves SLA status for a specific ticket
// @Summary Get ticket SLA status
// @Description Retrieve detailed SLA status information for a specific ticket
// @Tags SLA Metrics
// @Accept json
// @Produce json
// @Param id path string true "Ticket ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/tickets/{id}/sla-status [get]
func (c *SLAMetricsController) GetTicketSLAStatus(ctx *fiber.Ctx) error {
	ticketID := ctx.Params("id")
	if ticketID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Ticket ID is required",
		})
	}

	// Get ticket
	ticket, err := c.ticketService.GetTicket(ctx.Context(), ticketID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Ticket not found",
		})
	}

	// Get SLA policy if assigned
	var policy *models.SLAPolicy
	if ticket.SLAPolicyID != nil {
		policy, err = c.slaService.GetPolicy(ctx.Context(), ticket.SLAPolicyID.Hex())
		if err != nil {
			// Policy not found or error - continue with nil policy
			policy = nil
		}
	}

	// Calculate SLA status
	status := c.slaService.CalculateSLAStatus(ctx.Context(), ticket, policy)

	return ctx.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"ticket_id":     ticketID,
			"ticket_number": ticket.TicketNumber,
			"sla_status":    status,
			"policy":        policy,
		},
	})
}
