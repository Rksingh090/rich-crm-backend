package ticket

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type SLAMetricsController struct {
	SLAService    SLAService
	TicketService TicketService
}

func NewSLAMetricsController(slaService SLAService, ticketService TicketService) *SLAMetricsController {
	return &SLAMetricsController{
		SLAService:    slaService,
		TicketService: ticketService,
	}
}

// GetOverview godoc
func (ctrl *SLAMetricsController) GetOverview(c *fiber.Ctx) error {
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	startDate, _ := time.Parse("2006-01-02", startDateStr)
	endDate, _ := time.Parse("2006-01-02", endDateStr)

	if startDate.IsZero() {
		startDate = time.Now().AddDate(0, 0, -30) // Default last 30 days
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	metrics, err := ctrl.SLAService.GetSLAMetrics(c.Context(), startDate, endDate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(metrics)
}

// GetViolations godoc
func (ctrl *SLAMetricsController) GetViolations(c *fiber.Ctx) error {
	violations, err := ctrl.SLAService.GetSLAViolations(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(violations)
}

// GetTrends godoc
func (ctrl *SLAMetricsController) GetTrends(c *fiber.Ctx) error {
	days := 7 // Default 7 days
	trends, err := ctrl.SLAService.GetSLATrends(c.Context(), days)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(trends)
}

// GetTicketSLAStatus godoc
func (ctrl *SLAMetricsController) GetTicketSLAStatus(c *fiber.Ctx) error {
	ticketID := c.Params("id")

	ticket, err := ctrl.TicketService.GetTicket(c.Context(), ticketID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Ticket not found",
		})
	}

	// Find the policy
	var policy *SLAPolicy
	if ticket.SLAPolicyID != nil {
		policy, _ = ctrl.SLAService.GetPolicy(c.Context(), ticket.SLAPolicyID.Hex())
	}

	status := ctrl.SLAService.CalculateSLAStatus(c.Context(), ticket, policy)
	return c.JSON(status)
}
