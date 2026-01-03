package ticket

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TicketController struct {
	TicketService     TicketService
	SLAService        SLAService
	EscalationService EscalationService
}

func NewTicketController(
	ticketService TicketService,
	slaService SLAService,
	escalationService EscalationService,
) *TicketController {
	return &TicketController{
		TicketService:     ticketService,
		SLAService:        slaService,
		EscalationService: escalationService,
	}
}

// CreateTicket godoc
func (ctrl *TicketController) CreateTicket(c *fiber.Ctx) error {
	var ticket Ticket
	if err := c.BodyParser(&ticket); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get user ID from context (set by auth middleware)
	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	if err := ctrl.TicketService.CreateTicket(c.Context(), &ticket, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Ticket created successfully",
		"data":    ticket,
	})
}

// ListTickets godoc
func (ctrl *TicketController) ListTickets(c *fiber.Ctx) error {
	page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.Query("limit", "10"), 10, 64)
	sortBy := c.Query("sort_by", "created_at")
	sortOrder := c.Query("order", "desc")

	// Build filters
	filters := make(map[string]interface{})
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if priority := c.Query("priority"); priority != "" {
		filters["priority"] = priority
	}
	if channel := c.Query("channel"); channel != "" {
		filters["channel"] = channel
	}
	if assignedTo := c.Query("assigned_to"); assignedTo != "" {
		filters["assigned_to"] = assignedTo
	}
	if customerID := c.Query("customer_id"); customerID != "" {
		filters["customer_id"] = customerID
	}
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	tickets, totalCount, err := ctrl.TicketService.ListTickets(c.Context(), filters, page, limit, sortBy, sortOrder)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": tickets,
		"meta": fiber.Map{
			"total": totalCount,
			"page":  page,
			"limit": limit,
		},
	})
}

// GetTicket godoc
func (ctrl *TicketController) GetTicket(c *fiber.Ctx) error {
	id := c.Params("id")

	ticket, err := ctrl.TicketService.GetTicket(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(ticket)
}

// UpdateTicket godoc
func (ctrl *TicketController) UpdateTicket(c *fiber.Ctx) error {
	id := c.Params("id")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	if err := ctrl.TicketService.UpdateTicket(c.Context(), id, updates, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Ticket updated successfully",
	})
}

// DeleteTicket godoc
func (ctrl *TicketController) DeleteTicket(c *fiber.Ctx) error {
	id := c.Params("id")

	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	if err := ctrl.TicketService.DeleteTicket(c.Context(), id, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Ticket deleted successfully",
	})
}

// UpdateStatus godoc
func (ctrl *TicketController) UpdateStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	var input struct {
		Status  string `json:"status"`
		Comment string `json:"comment"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	status := TicketStatus(input.Status)
	if err := ctrl.TicketService.UpdateStatus(c.Context(), id, status, input.Comment, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Status updated successfully",
	})
}

// AssignTicket godoc
func (ctrl *TicketController) AssignTicket(c *fiber.Ctx) error {
	id := c.Params("id")

	var input struct {
		AssignedTo string `json:"assigned_to"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	assignedTo, err := primitive.ObjectIDFromHex(input.AssignedTo)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	if err := ctrl.TicketService.AssignTicket(c.Context(), id, assignedTo, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Ticket assigned successfully",
	})
}

// AddComment godoc
func (ctrl *TicketController) AddComment(c *fiber.Ctx) error {
	id := c.Params("id")

	var comment TicketComment
	if err := c.BodyParser(&comment); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}
	comment.CreatedBy = userID

	if err := ctrl.TicketService.AddComment(c.Context(), id, &comment); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Comment added successfully",
		"data":    comment,
	})
}

// ListComments godoc
func (ctrl *TicketController) ListComments(c *fiber.Ctx) error {
	id := c.Params("id")

	comments, err := ctrl.TicketService.ListComments(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": comments,
	})
}

// GetMyTickets godoc
func (ctrl *TicketController) GetMyTickets(c *fiber.Ctx) error {
	page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.Query("limit", "10"), 10, 64)

	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	tickets, totalCount, err := ctrl.TicketService.GetMyTickets(c.Context(), userID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": tickets,
		"meta": fiber.Map{
			"total": totalCount,
			"page":  page,
			"limit": limit,
		},
	})
}

// GetCustomerTickets godoc
func (ctrl *TicketController) GetCustomerTickets(c *fiber.Ctx) error {
	customerIDStr := c.Params("customerId")
	customerID, err := primitive.ObjectIDFromHex(customerIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid customer ID",
		})
	}

	page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.Query("limit", "10"), 10, 64)

	tickets, totalCount, err := ctrl.TicketService.GetCustomerTickets(c.Context(), customerID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": tickets,
		"meta": fiber.Map{
			"total": totalCount,
			"page":  page,
			"limit": limit,
		},
	})
}

// SLA Policy Controllers

// CreateSLAPolicy godoc
func (ctrl *TicketController) CreateSLAPolicy(c *fiber.Ctx) error {
	var policy SLAPolicy
	if err := c.BodyParser(&policy); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.SLAService.CreatePolicy(c.Context(), &policy); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "SLA policy created successfully",
		"data":    policy,
	})
}

// ListSLAPolicies godoc
func (ctrl *TicketController) ListSLAPolicies(c *fiber.Ctx) error {
	policies, err := ctrl.SLAService.ListPolicies(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": policies,
	})
}

// GetSLAPolicy godoc
func (ctrl *TicketController) GetSLAPolicy(c *fiber.Ctx) error {
	id := c.Params("id")

	policy, err := ctrl.SLAService.GetPolicy(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(policy)
}

// UpdateSLAPolicy godoc
func (ctrl *TicketController) UpdateSLAPolicy(c *fiber.Ctx) error {
	id := c.Params("id")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.SLAService.UpdatePolicy(c.Context(), id, updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "SLA policy updated successfully",
	})
}

// DeleteSLAPolicy godoc
func (ctrl *TicketController) DeleteSLAPolicy(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.SLAService.DeletePolicy(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "SLA policy deleted successfully",
	})
}

// Escalation Rule Controllers

// CreateEscalationRule godoc
func (ctrl *TicketController) CreateEscalationRule(c *fiber.Ctx) error {
	var rule EscalationRule
	if err := c.BodyParser(&rule); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.EscalationService.CreateRule(c.Context(), &rule); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Escalation rule created successfully",
		"data":    rule,
	})
}

// ListEscalationRules godoc
func (ctrl *TicketController) ListEscalationRules(c *fiber.Ctx) error {
	rules, err := ctrl.EscalationService.ListRules(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": rules,
	})
}

// GetEscalationRule godoc
func (ctrl *TicketController) GetEscalationRule(c *fiber.Ctx) error {
	id := c.Params("id")

	rule, err := ctrl.EscalationService.GetRule(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(rule)
}

// UpdateEscalationRule godoc
func (ctrl *TicketController) UpdateEscalationRule(c *fiber.Ctx) error {
	id := c.Params("id")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.EscalationService.UpdateRule(c.Context(), id, updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Escalation rule updated successfully",
	})
}

// DeleteEscalationRule godoc
func (ctrl *TicketController) DeleteEscalationRule(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.EscalationService.DeleteRule(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Escalation rule deleted successfully",
	})
}
