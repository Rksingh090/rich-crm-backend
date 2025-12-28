package controllers

import (
	"strconv"

	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TicketController struct {
	TicketService     service.TicketService
	SLAService        service.SLAService
	EscalationService service.EscalationService
}

func NewTicketController(
	ticketService service.TicketService,
	slaService service.SLAService,
	escalationService service.EscalationService,
) *TicketController {
	return &TicketController{
		TicketService:     ticketService,
		SLAService:        slaService,
		EscalationService: escalationService,
	}
}

// CreateTicket godoc
// @Summary      Create a new ticket
// @Description  Create a new support ticket
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        input body models.Ticket true "Ticket Data"
// @Success      201  {object} map[string]interface{}
// @Failure      400  {string} string "Invalid request"
// @Router       /api/tickets [post]
func (ctrl *TicketController) CreateTicket(c *fiber.Ctx) error {
	var ticket models.Ticket
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
// @Summary      List tickets
// @Description  Get tickets with filtering and pagination
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        page     query int    false "Page number (default 1)"
// @Param        limit    query int    false "Records per page (default 10)"
// @Param        status   query string false "Filter by status"
// @Param        priority query string false "Filter by priority"
// @Param        channel  query string false "Filter by channel"
// @Param        search   query string false "Search by ticket number, subject, or customer"
// @Success      200   {array}  models.Ticket
// @Failure      400   {string} string "Invalid input"
// @Router       /api/tickets [get]
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
// @Summary      Get a ticket
// @Description  Get a ticket by ID
// @Tags         tickets
// @Produce      json
// @Param        id path string true "Ticket ID"
// @Success      200 {object} models.Ticket
// @Failure      404 {string} string "Not Found"
// @Router       /api/tickets/{id} [get]
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
// @Summary      Update a ticket
// @Description  Update a ticket by ID
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        id    path string true "Ticket ID"
// @Param        input body map[string]interface{} true "Update Data"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid input"
// @Router       /api/tickets/{id} [put]
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
// @Summary      Delete a ticket
// @Description  Delete a ticket by ID
// @Tags         tickets
// @Produce      json
// @Param        id path string true "Ticket ID"
// @Success      200 {object} map[string]string
// @Failure      400 {string} string "Invalid input"
// @Router       /api/tickets/{id} [delete]
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
// @Summary      Update ticket status
// @Description  Update the status of a ticket
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        id    path string true "Ticket ID"
// @Param        input body map[string]interface{} true "Status Data"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid input"
// @Router       /api/tickets/{id}/status [patch]
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

	status := models.TicketStatus(input.Status)
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
// @Summary      Assign ticket
// @Description  Assign a ticket to a user
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        id    path string true "Ticket ID"
// @Param        input body map[string]interface{} true "Assignment Data"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid input"
// @Router       /api/tickets/{id}/assign [patch]
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
// @Summary      Add comment to ticket
// @Description  Add a comment or note to a ticket
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        id    path string true "Ticket ID"
// @Param        input body models.TicketComment true "Comment Data"
// @Success      201  {object} map[string]interface{}
// @Failure      400  {string} string "Invalid input"
// @Router       /api/tickets/{id}/comments [post]
func (ctrl *TicketController) AddComment(c *fiber.Ctx) error {
	id := c.Params("id")

	var comment models.TicketComment
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
// @Summary      List ticket comments
// @Description  Get all comments for a ticket
// @Tags         tickets
// @Produce      json
// @Param        id path string true "Ticket ID"
// @Success      200 {array} models.TicketComment
// @Failure      400 {string} string "Invalid input"
// @Router       /api/tickets/{id}/comments [get]
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
// @Summary      Get my tickets
// @Description  Get tickets assigned to the current user
// @Tags         tickets
// @Produce      json
// @Param        page  query int false "Page number (default 1)"
// @Param        limit query int false "Records per page (default 10)"
// @Success      200 {array} models.Ticket
// @Failure      400 {string} string "Invalid input"
// @Router       /api/tickets/my [get]
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
// @Summary      Get customer tickets
// @Description  Get all tickets for a specific customer
// @Tags         tickets
// @Produce      json
// @Param        customerId path string true "Customer ID"
// @Param        page       query int false "Page number (default 1)"
// @Param        limit      query int false "Records per page (default 10)"
// @Success      200 {array} models.Ticket
// @Failure      400 {string} string "Invalid input"
// @Router       /api/tickets/customer/{customerId} [get]
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
// @Summary      Create SLA policy
// @Description  Create a new SLA policy
// @Tags         sla-policies
// @Accept       json
// @Produce      json
// @Param        input body models.SLAPolicy true "SLA Policy Data"
// @Success      201  {object} map[string]interface{}
// @Failure      400  {string} string "Invalid request"
// @Router       /api/sla-policies [post]
func (ctrl *TicketController) CreateSLAPolicy(c *fiber.Ctx) error {
	var policy models.SLAPolicy
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
// @Summary      List SLA policies
// @Description  Get all SLA policies
// @Tags         sla-policies
// @Produce      json
// @Success      200 {array} models.SLAPolicy
// @Router       /api/sla-policies [get]
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
// @Summary      Get SLA policy
// @Description  Get an SLA policy by ID
// @Tags         sla-policies
// @Produce      json
// @Param        id path string true "Policy ID"
// @Success      200 {object} models.SLAPolicy
// @Failure      404 {string} string "Not Found"
// @Router       /api/sla-policies/{id} [get]
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
// @Summary      Update SLA policy
// @Description  Update an SLA policy by ID
// @Tags         sla-policies
// @Accept       json
// @Produce      json
// @Param        id    path string true "Policy ID"
// @Param        input body map[string]interface{} true "Update Data"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid input"
// @Router       /api/sla-policies/{id} [put]
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
// @Summary      Delete SLA policy
// @Description  Delete an SLA policy by ID
// @Tags         sla-policies
// @Produce      json
// @Param        id path string true "Policy ID"
// @Success      200 {object} map[string]string
// @Failure      400 {string} string "Invalid input"
// @Router       /api/sla-policies/{id} [delete]
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
// @Summary      Create escalation rule
// @Description  Create a new escalation rule
// @Tags         escalation-rules
// @Accept       json
// @Produce      json
// @Param        input body models.EscalationRule true "Escalation Rule Data"
// @Success      201  {object} map[string]interface{}
// @Failure      400  {string} string "Invalid request"
// @Router       /api/escalation-rules [post]
func (ctrl *TicketController) CreateEscalationRule(c *fiber.Ctx) error {
	var rule models.EscalationRule
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
// @Summary      List escalation rules
// @Description  Get all escalation rules
// @Tags         escalation-rules
// @Produce      json
// @Success      200 {array} models.EscalationRule
// @Router       /api/escalation-rules [get]
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
// @Summary      Get escalation rule
// @Description  Get an escalation rule by ID
// @Tags         escalation-rules
// @Produce      json
// @Param        id path string true "Rule ID"
// @Success      200 {object} models.EscalationRule
// @Failure      404 {string} string "Not Found"
// @Router       /api/escalation-rules/{id} [get]
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
// @Summary      Update escalation rule
// @Description  Update an escalation rule by ID
// @Tags         escalation-rules
// @Accept       json
// @Produce      json
// @Param        id    path string true "Rule ID"
// @Param        input body map[string]interface{} true "Update Data"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid input"
// @Router       /api/escalation-rules/{id} [put]
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
// @Summary      Delete escalation rule
// @Description  Delete an escalation rule by ID
// @Tags         escalation-rules
// @Produce      json
// @Param        id path string true "Rule ID"
// @Success      200 {object} map[string]string
// @Failure      400 {string} string "Invalid input"
// @Router       /api/escalation-rules/{id} [delete]
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
