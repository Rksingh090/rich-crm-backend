package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type TicketApi struct {
	controller *controllers.TicketController
	config     *config.Config
}

func NewTicketApi(controller *controllers.TicketController, config *config.Config) *TicketApi {
	return &TicketApi{
		controller: controller,
		config:     config,
	}
}

// Setup registers all ticket-related routes
func (h *TicketApi) Setup(app *fiber.App) {
	// Ticket routes
	tickets := app.Group("/api/tickets", middleware.AuthMiddleware(h.config.SkipAuth))

	// Ticket CRUD
	tickets.Post("/", h.controller.CreateTicket)
	tickets.Get("/", h.controller.ListTickets)
	tickets.Get("/my", h.controller.GetMyTickets)
	tickets.Get("/customer/:customerId", h.controller.GetCustomerTickets)
	tickets.Get("/:id", h.controller.GetTicket)
	tickets.Put("/:id", h.controller.UpdateTicket)
	tickets.Delete("/:id", h.controller.DeleteTicket)

	// Ticket actions
	tickets.Patch("/:id/status", h.controller.UpdateStatus)
	tickets.Patch("/:id/assign", h.controller.AssignTicket)

	// Comments
	tickets.Post("/:id/comments", h.controller.AddComment)
	tickets.Get("/:id/comments", h.controller.ListComments)

	// SLA Policy routes
	slaPolicies := app.Group("/api/sla-policies", middleware.AuthMiddleware(h.config.SkipAuth))
	slaPolicies.Post("/", h.controller.CreateSLAPolicy)
	slaPolicies.Get("/", h.controller.ListSLAPolicies)
	slaPolicies.Get("/:id", h.controller.GetSLAPolicy)
	slaPolicies.Put("/:id", h.controller.UpdateSLAPolicy)
	slaPolicies.Delete("/:id", h.controller.DeleteSLAPolicy)

	// Escalation Rule routes
	escalationRules := app.Group("/api/escalation-rules", middleware.AuthMiddleware(h.config.SkipAuth))
	escalationRules.Post("/", h.controller.CreateEscalationRule)
	escalationRules.Get("/", h.controller.ListEscalationRules)
	escalationRules.Get("/:id", h.controller.GetEscalationRule)
	escalationRules.Put("/:id", h.controller.UpdateEscalationRule)
	escalationRules.Delete("/:id", h.controller.DeleteEscalationRule)
}
