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
// CreateTicket godoc
// @Summary Create ticket
// @Description Create a new support ticket
// @Tags tickets
// @Accept json
// @Produce json
// @Param ticket body Ticket true "Ticket Details"
// @Success 201 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/tickets [post]
func (ctrl *TicketController) CreateTicket(c *fiber.Ctx) error {
	var ticket Ticket
	if err := c.BodyParser(&ticket); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get user ID from context (set by auth middleware)
	userIDStr, ok := c.Locals("user_id").(string)
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

	if err := ctrl.TicketService.CreateTicket(c.UserContext(), &ticket, userID); err != nil {
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
// ListTickets godoc
// @Summary List tickets
// @Description List tickets with filtering and pagination
// @Tags tickets
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param status query string false "Filter by status"
// @Param priority query string false "Filter by priority"
// @Param channel query string false "Filter by channel"
// @Param assigned_to query string false "Filter by assignee"
// @Param search query string false "Search query"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/tickets [get]
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

	tickets, totalCount, err := ctrl.TicketService.ListTickets(c.UserContext(), filters, page, limit, sortBy, sortOrder)
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
// GetTicket godoc
// @Summary Get ticket
// @Description Get a ticket by ID
// @Tags tickets
// @Produce json
// @Param id path string true "Ticket ID"
// @Success 200 {object} Ticket
// @Failure 404 {object} map[string]interface{}
// @Router /api/tickets/{id} [get]
func (ctrl *TicketController) GetTicket(c *fiber.Ctx) error {
	id := c.Params("id")

	ticket, err := ctrl.TicketService.GetTicket(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(ticket)
}

// UpdateTicket godoc
// UpdateTicket godoc
// @Summary Update ticket
// @Description Update an existing ticket
// @Tags tickets
// @Accept json
// @Produce json
// @Param id path string true "Ticket ID"
// @Param updates body map[string]interface{} true "Ticket Updates"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/tickets/{id} [put]
func (ctrl *TicketController) UpdateTicket(c *fiber.Ctx) error {
	id := c.Params("id")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userIDStr, ok := c.Locals("user_id").(string)
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

	if err := ctrl.TicketService.UpdateTicket(c.UserContext(), id, updates, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Ticket updated successfully",
	})
}

// DeleteTicket godoc
// DeleteTicket godoc
// @Summary Delete ticket
// @Description Delete a ticket by ID
// @Tags tickets
// @Param id path string true "Ticket ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/tickets/{id} [delete]
func (ctrl *TicketController) DeleteTicket(c *fiber.Ctx) error {
	id := c.Params("id")

	userIDStr, ok := c.Locals("user_id").(string)
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

	if err := ctrl.TicketService.DeleteTicket(c.UserContext(), id, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Ticket deleted successfully",
	})
}

// UpdateStatus godoc
// UpdateStatus godoc
// @Summary Update ticket status
// @Description Update the status of a ticket with a comment
// @Tags tickets
// @Accept json
// @Produce json
// @Param id path string true "Ticket ID"
// @Param body body map[string]string true "Status Update {status, comment}"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/tickets/{id}/status [put]
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

	userIDStr, ok := c.Locals("user_id").(string)
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
	if err := ctrl.TicketService.UpdateStatus(c.UserContext(), id, status, input.Comment, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Status updated successfully",
	})
}

// AssignTicket godoc
// AssignTicket godoc
// @Summary Assign ticket
// @Description Assign a ticket to a user
// @Tags tickets
// @Accept json
// @Produce json
// @Param id path string true "Ticket ID"
// @Param body body map[string]string true "Assignment {assigned_to}"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/tickets/{id}/assign [put]
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

	userIDStr, ok := c.Locals("user_id").(string)
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

	if err := ctrl.TicketService.AssignTicket(c.UserContext(), id, assignedTo, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Ticket assigned successfully",
	})
}

// AddComment godoc
// AddComment godoc
// @Summary Add comment
// @Description Add a comment to a ticket
// @Tags tickets
// @Accept json
// @Produce json
// @Param id path string true "Ticket ID"
// @Param comment body TicketComment true "Comment Details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/tickets/{id}/comments [post]
func (ctrl *TicketController) AddComment(c *fiber.Ctx) error {
	id := c.Params("id")

	var comment TicketComment
	if err := c.BodyParser(&comment); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userIDStr, ok := c.Locals("user_id").(string)
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

	if err := ctrl.TicketService.AddComment(c.UserContext(), id, &comment); err != nil {
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
// ListComments godoc
// @Summary List comments
// @Description List comments for a ticket
// @Tags tickets
// @Produce json
// @Param id path string true "Ticket ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/tickets/{id}/comments [get]
func (ctrl *TicketController) ListComments(c *fiber.Ctx) error {
	id := c.Params("id")

	comments, err := ctrl.TicketService.ListComments(c.UserContext(), id)
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
// GetMyTickets godoc
// @Summary Get my tickets
// @Description List tickets assigned to the current user
// @Tags tickets
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/tickets/my-tickets [get]
func (ctrl *TicketController) GetMyTickets(c *fiber.Ctx) error {
	page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.Query("limit", "10"), 10, 64)

	userIDStr, ok := c.Locals("user_id").(string)
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

	tickets, totalCount, err := ctrl.TicketService.GetMyTickets(c.UserContext(), userID, page, limit)
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
// GetCustomerTickets godoc
// @Summary Get customer tickets
// @Description List tickets for a specific customer
// @Tags tickets
// @Produce json
// @Param customerId path string true "Customer ID"
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/tickets/customer/{customerId} [get]
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

	tickets, totalCount, err := ctrl.TicketService.GetCustomerTickets(c.UserContext(), customerID, page, limit)
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
// CreateSLAPolicy godoc
// @Summary Create SLA policy
// @Description Create a new SLA policy
// @Tags sla
// @Accept json
// @Produce json
// @Param policy body SLAPolicy true "SLA Policy"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/sla/policies [post]
func (ctrl *TicketController) CreateSLAPolicy(c *fiber.Ctx) error {
	var policy SLAPolicy
	if err := c.BodyParser(&policy); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.SLAService.CreatePolicy(c.UserContext(), &policy); err != nil {
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
// ListSLAPolicies godoc
// @Summary List SLA policies
// @Description List all SLA policies
// @Tags sla
// @Produce json
// @Success 200 {array} SLAPolicy
// @Failure 500 {object} map[string]interface{}
// @Router /api/sla/policies [get]
func (ctrl *TicketController) ListSLAPolicies(c *fiber.Ctx) error {
	policies, err := ctrl.SLAService.ListPolicies(c.UserContext())
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
// GetSLAPolicy godoc
// @Summary Get SLA policy
// @Description Get an SLA policy by ID
// @Tags sla
// @Produce json
// @Param id path string true "Policy ID"
// @Success 200 {object} SLAPolicy
// @Failure 404 {object} map[string]interface{}
// @Router /api/sla/policies/{id} [get]
func (ctrl *TicketController) GetSLAPolicy(c *fiber.Ctx) error {
	id := c.Params("id")

	policy, err := ctrl.SLAService.GetPolicy(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(policy)
}

// UpdateSLAPolicy godoc
// UpdateSLAPolicy godoc
// @Summary Update SLA policy
// @Description Update an existing SLA policy
// @Tags sla
// @Accept json
// @Produce json
// @Param id path string true "Policy ID"
// @Param updates body map[string]interface{} true "Policy Updates"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/sla/policies/{id} [put]
func (ctrl *TicketController) UpdateSLAPolicy(c *fiber.Ctx) error {
	id := c.Params("id")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.SLAService.UpdatePolicy(c.UserContext(), id, updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "SLA policy updated successfully",
	})
}

// DeleteSLAPolicy godoc
// DeleteSLAPolicy godoc
// @Summary Delete SLA policy
// @Description Delete an SLA policy by ID
// @Tags sla
// @Param id path string true "Policy ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/sla/policies/{id} [delete]
func (ctrl *TicketController) DeleteSLAPolicy(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.SLAService.DeletePolicy(c.UserContext(), id); err != nil {
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
// CreateEscalationRule godoc
// @Summary Create escalation rule
// @Description Create a new escalation rule
// @Tags escalation
// @Accept json
// @Produce json
// @Param rule body EscalationRule true "Escalation Rule"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/escalation/rules [post]
func (ctrl *TicketController) CreateEscalationRule(c *fiber.Ctx) error {
	var rule EscalationRule
	if err := c.BodyParser(&rule); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.EscalationService.CreateRule(c.UserContext(), &rule); err != nil {
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
// ListEscalationRules godoc
// @Summary List escalation rules
// @Description List all escalation rules
// @Tags escalation
// @Produce json
// @Success 200 {array} EscalationRule
// @Failure 500 {object} map[string]interface{}
// @Router /api/escalation/rules [get]
func (ctrl *TicketController) ListEscalationRules(c *fiber.Ctx) error {
	rules, err := ctrl.EscalationService.ListRules(c.UserContext())
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
// GetEscalationRule godoc
// @Summary Get escalation rule
// @Description Get an escalation rule by ID
// @Tags escalation
// @Produce json
// @Param id path string true "Rule ID"
// @Success 200 {object} EscalationRule
// @Failure 404 {object} map[string]interface{}
// @Router /api/escalation/rules/{id} [get]
func (ctrl *TicketController) GetEscalationRule(c *fiber.Ctx) error {
	id := c.Params("id")

	rule, err := ctrl.EscalationService.GetRule(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(rule)
}

// UpdateEscalationRule godoc
// UpdateEscalationRule godoc
// @Summary Update escalation rule
// @Description Update an existing escalation rule
// @Tags escalation
// @Accept json
// @Produce json
// @Param id path string true "Rule ID"
// @Param updates body map[string]interface{} true "Rule Updates"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/escalation/rules/{id} [put]
func (ctrl *TicketController) UpdateEscalationRule(c *fiber.Ctx) error {
	id := c.Params("id")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.EscalationService.UpdateRule(c.UserContext(), id, updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Escalation rule updated successfully",
	})
}

// DeleteEscalationRule godoc
// DeleteEscalationRule godoc
// @Summary Delete escalation rule
// @Description Delete an escalation rule by ID
// @Tags escalation
// @Param id path string true "Rule ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/escalation/rules/{id} [delete]
func (ctrl *TicketController) DeleteEscalationRule(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.EscalationService.DeleteRule(c.UserContext(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Escalation rule deleted successfully",
	})
}
