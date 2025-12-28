package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TicketService defines the interface for ticket business logic
type TicketService interface {
	CreateTicket(ctx context.Context, ticket *models.Ticket, createdBy primitive.ObjectID) error
	GetTicket(ctx context.Context, id string) (*models.Ticket, error)
	ListTickets(ctx context.Context, filters map[string]interface{}, page, limit int64, sortBy, sortOrder string) ([]models.Ticket, int64, error)
	UpdateTicket(ctx context.Context, id string, updates map[string]interface{}, updatedBy primitive.ObjectID) error
	DeleteTicket(ctx context.Context, id string, deletedBy primitive.ObjectID) error

	// Status Management
	UpdateStatus(ctx context.Context, id string, status models.TicketStatus, comment string, changedBy primitive.ObjectID) error
	GetStatusHistory(ctx context.Context, id string) ([]models.StatusHistoryEntry, error)

	// Assignment
	AssignTicket(ctx context.Context, id string, assignedTo primitive.ObjectID, assignedBy primitive.ObjectID) error
	UnassignTicket(ctx context.Context, id string, unassignedBy primitive.ObjectID) error
	GetMyTickets(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]models.Ticket, int64, error)
	GetCustomerTickets(ctx context.Context, customerID primitive.ObjectID, page, limit int64) ([]models.Ticket, int64, error)

	// Comments
	AddComment(ctx context.Context, ticketID string, comment *models.TicketComment) error
	ListComments(ctx context.Context, ticketID string) ([]models.TicketComment, error)

	// SLA Management
	CalculateDueDates(ctx context.Context, ticket *models.Ticket) error
	CheckSLABreach(ctx context.Context, ticketID string) (bool, error)
	GetOverdueSLATickets(ctx context.Context) ([]models.Ticket, error)

	// Multi-Channel
	CreateTicketFromEmail(ctx context.Context, subject, description, customerEmail, customerName string, metadata map[string]interface{}) error
	CreateTicketFromChat(ctx context.Context, subject, description, customerEmail, customerName string, metadata map[string]interface{}) error
	CreateTicketFromPortal(ctx context.Context, ticket *models.Ticket, createdBy primitive.ObjectID) error
}

// TicketServiceImpl implements TicketService
type TicketServiceImpl struct {
	TicketRepo    repository.TicketRepository
	SLAPolicyRepo repository.SLAPolicyRepository
	CommentRepo   repository.TicketCommentRepository
	AuditService  AuditService
}

// NewTicketService creates a new ticket service
func NewTicketService(
	ticketRepo repository.TicketRepository,
	slaPolicyRepo repository.SLAPolicyRepository,
	commentRepo repository.TicketCommentRepository,
	auditService AuditService,
) TicketService {
	return &TicketServiceImpl{
		TicketRepo:    ticketRepo,
		SLAPolicyRepo: slaPolicyRepo,
		CommentRepo:   commentRepo,
		AuditService:  auditService,
	}
}

// CreateTicket creates a new ticket
func (s *TicketServiceImpl) CreateTicket(ctx context.Context, ticket *models.Ticket, createdBy primitive.ObjectID) error {
	// Generate ticket number
	ticketNumber, err := s.TicketRepo.GetNextTicketNumber(ctx)
	if err != nil {
		return err
	}
	ticket.TicketNumber = ticketNumber

	// Set initial status
	if ticket.Status == "" {
		ticket.Status = models.TicketStatusNew
	}

	// Initialize status history
	ticket.StatusHistory = []models.StatusHistoryEntry{
		{
			Status:    ticket.Status,
			ChangedBy: createdBy,
			ChangedAt: time.Now(),
			Comment:   "Ticket created",
		},
	}

	// Calculate SLA due dates
	if err := s.CalculateDueDates(ctx, ticket); err != nil {
		return err
	}

	// Create ticket
	if err := s.TicketRepo.Create(ctx, ticket); err != nil {
		return err
	}

	// Audit log
	changes := map[string]models.Change{
		"ticket_number": {Old: nil, New: ticket.TicketNumber},
		"subject":       {Old: nil, New: ticket.Subject},
		"priority":      {Old: nil, New: ticket.Priority},
		"status":        {Old: nil, New: ticket.Status},
	}
	s.AuditService.LogChange(ctx, models.AuditActionCreate, "tickets", ticket.ID.Hex(), changes)

	return nil
}

// GetTicket retrieves a ticket by ID
func (s *TicketServiceImpl) GetTicket(ctx context.Context, id string) (*models.Ticket, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid ticket ID")
	}

	return s.TicketRepo.FindByID(ctx, objID)
}

// ListTickets retrieves tickets with filtering and pagination
func (s *TicketServiceImpl) ListTickets(ctx context.Context, filters map[string]interface{}, page, limit int64, sortBy, sortOrder string) ([]models.Ticket, int64, error) {
	// Build MongoDB filter
	filter := bson.M{}

	if status, ok := filters["status"].(string); ok && status != "" {
		filter["status"] = status
	}

	if priority, ok := filters["priority"].(string); ok && priority != "" {
		filter["priority"] = priority
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		filter["channel"] = channel
	}

	if assignedTo, ok := filters["assigned_to"].(string); ok && assignedTo != "" {
		objID, err := primitive.ObjectIDFromHex(assignedTo)
		if err == nil {
			filter["assigned_to"] = objID
		}
	}

	if customerID, ok := filters["customer_id"].(string); ok && customerID != "" {
		objID, err := primitive.ObjectIDFromHex(customerID)
		if err == nil {
			filter["customer_id"] = objID
		}
	}

	if search, ok := filters["search"].(string); ok && search != "" {
		filter["$or"] = []bson.M{
			{"ticket_number": bson.M{"$regex": search, "$options": "i"}},
			{"subject": bson.M{"$regex": search, "$options": "i"}},
			{"customer_name": bson.M{"$regex": search, "$options": "i"}},
			{"customer_email": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	return s.TicketRepo.FindAll(ctx, filter, page, limit, sortBy, sortOrder)
}

// UpdateTicket updates a ticket
func (s *TicketServiceImpl) UpdateTicket(ctx context.Context, id string, updates map[string]interface{}, updatedBy primitive.ObjectID) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid ticket ID")
	}

	// Get existing ticket for audit
	oldTicket, err := s.TicketRepo.FindByID(ctx, objID)
	if err != nil {
		return err
	}

	// Convert to bson.M
	bsonUpdates := bson.M{}
	for k, v := range updates {
		bsonUpdates[k] = v
	}

	// Update ticket
	if err := s.TicketRepo.Update(ctx, objID, bsonUpdates); err != nil {
		return err
	}

	// Audit log - build changes map
	changes := make(map[string]models.Change)
	if subject, ok := updates["subject"]; ok {
		changes["subject"] = models.Change{Old: oldTicket.Subject, New: subject}
	}
	if description, ok := updates["description"]; ok {
		changes["description"] = models.Change{Old: oldTicket.Description, New: description}
	}
	if priority, ok := updates["priority"]; ok {
		changes["priority"] = models.Change{Old: oldTicket.Priority, New: priority}
	}

	if len(changes) > 0 {
		s.AuditService.LogChange(ctx, models.AuditActionUpdate, "tickets", objID.Hex(), changes)
	}

	return nil
}

// DeleteTicket deletes a ticket
func (s *TicketServiceImpl) DeleteTicket(ctx context.Context, id string, deletedBy primitive.ObjectID) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid ticket ID")
	}

	// Get ticket for audit
	ticket, err := s.TicketRepo.FindByID(ctx, objID)
	if err != nil {
		return err
	}

	// Delete ticket
	if err := s.TicketRepo.Delete(ctx, objID); err != nil {
		return err
	}

	// Audit log
	changes := map[string]models.Change{
		"ticket_number": {Old: ticket.TicketNumber, New: nil},
		"subject":       {Old: ticket.Subject, New: nil},
	}
	s.AuditService.LogChange(ctx, models.AuditActionDelete, "tickets", objID.Hex(), changes)

	return nil
}

// UpdateStatus updates the ticket status
func (s *TicketServiceImpl) UpdateStatus(ctx context.Context, id string, status models.TicketStatus, comment string, changedBy primitive.ObjectID) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid ticket ID")
	}

	// Get old status for audit
	oldTicket, err := s.TicketRepo.FindByID(ctx, objID)
	if err != nil {
		return err
	}

	// Validate status transition (basic validation)
	validStatuses := map[models.TicketStatus]bool{
		models.TicketStatusNew:      true,
		models.TicketStatusOpen:     true,
		models.TicketStatusPending:  true,
		models.TicketStatusResolved: true,
		models.TicketStatusClosed:   true,
	}

	if !validStatuses[status] {
		return errors.New("invalid status")
	}

	// Create history entry
	historyEntry := models.StatusHistoryEntry{
		Status:    status,
		ChangedBy: changedBy,
		ChangedAt: time.Now(),
		Comment:   comment,
	}

	// Update status
	if err := s.TicketRepo.UpdateStatus(ctx, objID, status, historyEntry); err != nil {
		return err
	}

	// Audit log
	changes := map[string]models.Change{
		"status": {Old: oldTicket.Status, New: status},
	}
	s.AuditService.LogChange(ctx, models.AuditActionUpdate, "tickets", objID.Hex(), changes)

	return nil
}

// GetStatusHistory retrieves the status history of a ticket
func (s *TicketServiceImpl) GetStatusHistory(ctx context.Context, id string) ([]models.StatusHistoryEntry, error) {
	ticket, err := s.GetTicket(ctx, id)
	if err != nil {
		return nil, err
	}

	return ticket.StatusHistory, nil
}

// AssignTicket assigns a ticket to a user
func (s *TicketServiceImpl) AssignTicket(ctx context.Context, id string, assignedTo primitive.ObjectID, assignedBy primitive.ObjectID) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid ticket ID")
	}

	// Get old assignment for audit
	oldTicket, err := s.TicketRepo.FindByID(ctx, objID)
	if err != nil {
		return err
	}

	updates := bson.M{
		"assigned_to": assignedTo,
	}

	if err := s.TicketRepo.Update(ctx, objID, updates); err != nil {
		return err
	}

	// Audit log
	var oldAssignee interface{} = nil
	if oldTicket.AssignedTo != nil {
		oldAssignee = oldTicket.AssignedTo.Hex()
	}
	changes := map[string]models.Change{
		"assigned_to": {Old: oldAssignee, New: assignedTo.Hex()},
	}
	s.AuditService.LogChange(ctx, models.AuditActionUpdate, "tickets", objID.Hex(), changes)

	return nil
}

// UnassignTicket removes the assignment from a ticket
func (s *TicketServiceImpl) UnassignTicket(ctx context.Context, id string, unassignedBy primitive.ObjectID) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid ticket ID")
	}

	// Get old assignment for audit
	oldTicket, err := s.TicketRepo.FindByID(ctx, objID)
	if err != nil {
		return err
	}

	updates := bson.M{
		"assigned_to": nil,
	}

	if err := s.TicketRepo.Update(ctx, objID, updates); err != nil {
		return err
	}

	// Audit log
	var oldAssignee interface{} = nil
	if oldTicket.AssignedTo != nil {
		oldAssignee = oldTicket.AssignedTo.Hex()
	}
	changes := map[string]models.Change{
		"assigned_to": {Old: oldAssignee, New: nil},
	}
	s.AuditService.LogChange(ctx, models.AuditActionUpdate, "tickets", objID.Hex(), changes)

	return nil
}

// GetMyTickets retrieves tickets assigned to a user
func (s *TicketServiceImpl) GetMyTickets(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]models.Ticket, int64, error) {
	return s.TicketRepo.FindByAssignee(ctx, userID, page, limit)
}

// GetCustomerTickets retrieves tickets for a customer
func (s *TicketServiceImpl) GetCustomerTickets(ctx context.Context, customerID primitive.ObjectID, page, limit int64) ([]models.Ticket, int64, error) {
	return s.TicketRepo.FindByCustomer(ctx, customerID, page, limit)
}

// AddComment adds a comment to a ticket
func (s *TicketServiceImpl) AddComment(ctx context.Context, ticketID string, comment *models.TicketComment) error {
	objID, err := primitive.ObjectIDFromHex(ticketID)
	if err != nil {
		return errors.New("invalid ticket ID")
	}

	// Verify ticket exists
	_, err = s.TicketRepo.FindByID(ctx, objID)
	if err != nil {
		return err
	}

	comment.TicketID = objID

	if err := s.CommentRepo.Create(ctx, comment); err != nil {
		return err
	}

	// Update first response time if this is the first response
	ticket, _ := s.TicketRepo.FindByID(ctx, objID)
	if ticket != nil && ticket.FirstResponseAt == nil && !comment.IsInternal {
		now := time.Now()
		s.TicketRepo.Update(ctx, objID, bson.M{"first_response_at": now})
	}

	// Audit log
	changes := map[string]models.Change{
		"comment_added": {Old: nil, New: comment.ID.Hex()},
	}
	s.AuditService.LogChange(ctx, models.AuditActionUpdate, "tickets", objID.Hex(), changes)

	return nil
}

// ListComments retrieves all comments for a ticket
func (s *TicketServiceImpl) ListComments(ctx context.Context, ticketID string) ([]models.TicketComment, error) {
	objID, err := primitive.ObjectIDFromHex(ticketID)
	if err != nil {
		return nil, errors.New("invalid ticket ID")
	}

	return s.CommentRepo.FindByTicketID(ctx, objID)
}

// CalculateDueDates calculates SLA due dates for a ticket
func (s *TicketServiceImpl) CalculateDueDates(ctx context.Context, ticket *models.Ticket) error {
	// Find SLA policy for the ticket's priority
	policy, err := s.SLAPolicyRepo.FindByPriority(ctx, ticket.Priority)
	if err != nil {
		return err
	}

	if policy == nil {
		// No SLA policy found, skip
		return nil
	}

	ticket.SLAPolicyID = &policy.ID

	now := time.Now()

	// Calculate response due date
	responseDue := now.Add(time.Duration(policy.ResponseTime) * time.Minute)
	ticket.ResponseDueDate = &responseDue

	// Calculate resolution due date
	resolutionDue := now.Add(time.Duration(policy.ResolutionTime) * time.Minute)
	ticket.DueDate = &resolutionDue

	return nil
}

// CheckSLABreach checks if a ticket has breached its SLA
func (s *TicketServiceImpl) CheckSLABreach(ctx context.Context, ticketID string) (bool, error) {
	ticket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return false, err
	}

	now := time.Now()

	// Check response SLA breach
	if ticket.ResponseDueDate != nil && ticket.FirstResponseAt == nil {
		if now.After(*ticket.ResponseDueDate) {
			return true, nil
		}
	}

	// Check resolution SLA breach
	if ticket.DueDate != nil {
		if ticket.Status != models.TicketStatusResolved && ticket.Status != models.TicketStatusClosed {
			if now.After(*ticket.DueDate) {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetOverdueSLATickets retrieves all tickets with SLA breaches
func (s *TicketServiceImpl) GetOverdueSLATickets(ctx context.Context) ([]models.Ticket, error) {
	return s.TicketRepo.FindOverdueSLA(ctx)
}

// CreateTicketFromEmail creates a ticket from an email
func (s *TicketServiceImpl) CreateTicketFromEmail(ctx context.Context, subject, description, customerEmail, customerName string, metadata map[string]interface{}) error {
	ticket := &models.Ticket{
		Subject:         subject,
		Description:     description,
		Channel:         models.TicketChannelEmail,
		ChannelMetadata: metadata,
		CustomerEmail:   customerEmail,
		CustomerName:    customerName,
		Priority:        models.TicketPriorityMedium, // Default priority
		Status:          models.TicketStatusNew,
	}

	// Use system user ID for automated ticket creation
	systemUserID := primitive.NewObjectID() // In production, use actual system user ID

	return s.CreateTicket(ctx, ticket, systemUserID)
}

// CreateTicketFromChat creates a ticket from a chat conversation
func (s *TicketServiceImpl) CreateTicketFromChat(ctx context.Context, subject, description, customerEmail, customerName string, metadata map[string]interface{}) error {
	ticket := &models.Ticket{
		Subject:         subject,
		Description:     description,
		Channel:         models.TicketChannelChat,
		ChannelMetadata: metadata,
		CustomerEmail:   customerEmail,
		CustomerName:    customerName,
		Priority:        models.TicketPriorityMedium,
		Status:          models.TicketStatusNew,
	}

	systemUserID := primitive.NewObjectID()

	return s.CreateTicket(ctx, ticket, systemUserID)
}

// CreateTicketFromPortal creates a ticket from the customer portal
func (s *TicketServiceImpl) CreateTicketFromPortal(ctx context.Context, ticket *models.Ticket, createdBy primitive.ObjectID) error {
	ticket.Channel = models.TicketChannelPortal
	ticket.CustomerID = &createdBy

	return s.CreateTicket(ctx, ticket, createdBy)
}

// SLAService defines the interface for SLA policy management
type SLAService interface {
	CreatePolicy(ctx context.Context, policy *models.SLAPolicy) error
	GetPolicy(ctx context.Context, id string) (*models.SLAPolicy, error)
	ListPolicies(ctx context.Context) ([]models.SLAPolicy, error)
	UpdatePolicy(ctx context.Context, id string, updates map[string]interface{}) error
	DeletePolicy(ctx context.Context, id string) error
}

// SLAServiceImpl implements SLAService
type SLAServiceImpl struct {
	SLAPolicyRepo repository.SLAPolicyRepository
}

// NewSLAService creates a new SLA service
func NewSLAService(slaPolicyRepo repository.SLAPolicyRepository) SLAService {
	return &SLAServiceImpl{
		SLAPolicyRepo: slaPolicyRepo,
	}
}

// CreatePolicy creates a new SLA policy
func (s *SLAServiceImpl) CreatePolicy(ctx context.Context, policy *models.SLAPolicy) error {
	return s.SLAPolicyRepo.Create(ctx, policy)
}

// GetPolicy retrieves an SLA policy by ID
func (s *SLAServiceImpl) GetPolicy(ctx context.Context, id string) (*models.SLAPolicy, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid policy ID")
	}

	return s.SLAPolicyRepo.FindByID(ctx, objID)
}

// ListPolicies retrieves all SLA policies
func (s *SLAServiceImpl) ListPolicies(ctx context.Context) ([]models.SLAPolicy, error) {
	return s.SLAPolicyRepo.FindAll(ctx)
}

// UpdatePolicy updates an SLA policy
func (s *SLAServiceImpl) UpdatePolicy(ctx context.Context, id string, updates map[string]interface{}) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid policy ID")
	}

	bsonUpdates := bson.M{}
	for k, v := range updates {
		bsonUpdates[k] = v
	}

	return s.SLAPolicyRepo.Update(ctx, objID, bsonUpdates)
}

// DeletePolicy deletes an SLA policy
func (s *SLAServiceImpl) DeletePolicy(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid policy ID")
	}

	return s.SLAPolicyRepo.Delete(ctx, objID)
}

// EscalationService defines the interface for escalation management
type EscalationService interface {
	ProcessEscalations(ctx context.Context) error
	EvaluateRules(ctx context.Context, ticket *models.Ticket) ([]models.EscalationRule, error)
	ExecuteEscalation(ctx context.Context, ticket *models.Ticket, rule *models.EscalationRule) error
	CreateRule(ctx context.Context, rule *models.EscalationRule) error
	GetRule(ctx context.Context, id string) (*models.EscalationRule, error)
	ListRules(ctx context.Context) ([]models.EscalationRule, error)
	UpdateRule(ctx context.Context, id string, updates map[string]interface{}) error
	DeleteRule(ctx context.Context, id string) error
}

// EscalationServiceImpl implements EscalationService
type EscalationServiceImpl struct {
	EscalationRuleRepo repository.EscalationRuleRepository
	TicketRepo         repository.TicketRepository
	AuditService       AuditService
}

// NewEscalationService creates a new escalation service
func NewEscalationService(
	escalationRuleRepo repository.EscalationRuleRepository,
	ticketRepo repository.TicketRepository,
	auditService AuditService,
) EscalationService {
	return &EscalationServiceImpl{
		EscalationRuleRepo: escalationRuleRepo,
		TicketRepo:         ticketRepo,
		AuditService:       auditService,
	}
}

// ProcessEscalations processes all tickets for escalation
func (s *EscalationServiceImpl) ProcessEscalations(ctx context.Context) error {
	// Get all open tickets
	tickets, _, err := s.TicketRepo.FindAll(ctx, bson.M{
		"status": bson.M{"$nin": []models.TicketStatus{models.TicketStatusResolved, models.TicketStatusClosed}},
	}, 1, 1000, "created_at", "asc")
	if err != nil {
		return err
	}

	// Evaluate each ticket against rules
	for _, ticket := range tickets {
		applicableRules, err := s.EvaluateRules(ctx, &ticket)
		if err != nil {
			continue
		}

		for _, rule := range applicableRules {
			s.ExecuteEscalation(ctx, &ticket, &rule)
		}
	}

	return nil
}

// EvaluateRules evaluates which escalation rules apply to a ticket
func (s *EscalationServiceImpl) EvaluateRules(ctx context.Context, ticket *models.Ticket) ([]models.EscalationRule, error) {
	rules, err := s.EscalationRuleRepo.FindActive(ctx)
	if err != nil {
		return nil, err
	}

	var applicableRules []models.EscalationRule
	now := time.Now()

	for _, rule := range rules {
		// Check priority match
		if rule.Priority != nil && *rule.Priority != ticket.Priority {
			continue
		}

		// Check status match
		if rule.Status != nil && *rule.Status != ticket.Status {
			continue
		}

		// Check time condition
		var referenceTime time.Time
		switch rule.ConditionType {
		case "sla_breach":
			if ticket.DueDate != nil && now.After(*ticket.DueDate) {
				applicableRules = append(applicableRules, rule)
			}
		case "no_response":
			if ticket.FirstResponseAt == nil {
				referenceTime = ticket.CreatedAt
				if now.Sub(referenceTime) > time.Duration(rule.EscalateAfter)*time.Minute {
					applicableRules = append(applicableRules, rule)
				}
			}
		case "no_update":
			referenceTime = ticket.UpdatedAt
			if now.Sub(referenceTime) > time.Duration(rule.EscalateAfter)*time.Minute {
				applicableRules = append(applicableRules, rule)
			}
		}
	}

	return applicableRules, nil
}

// ExecuteEscalation executes an escalation action
func (s *EscalationServiceImpl) ExecuteEscalation(ctx context.Context, ticket *models.Ticket, rule *models.EscalationRule) error {
	// Create escalation history entry
	escalationEntry := models.EscalationHistoryEntry{
		Level:       ticket.EscalationLevel + 1,
		EscalatedTo: rule.EscalateTo,
		EscalatedAt: time.Now(),
		Reason:      fmt.Sprintf("Escalated by rule: %s", rule.Name),
		RuleID:      rule.ID,
	}

	// Update ticket
	updates := bson.M{
		"escalation_level": ticket.EscalationLevel + 1,
		"escalated_to":     rule.EscalateTo,
	}

	if err := s.TicketRepo.Update(ctx, ticket.ID, updates); err != nil {
		return err
	}

	// Add to escalation history
	s.TicketRepo.Update(ctx, ticket.ID, bson.M{
		"$push": bson.M{"escalation_history": escalationEntry},
	})

	// Audit log
	changes := map[string]models.Change{
		"escalation_level": {Old: ticket.EscalationLevel, New: ticket.EscalationLevel + 1},
		"escalated_to":     {Old: nil, New: rule.EscalateTo.Hex()},
	}
	s.AuditService.LogChange(ctx, models.AuditActionUpdate, "tickets", ticket.ID.Hex(), changes)

	// TODO: Send notifications to escalated_to user and notify_emails

	return nil
}

// CreateRule creates a new escalation rule
func (s *EscalationServiceImpl) CreateRule(ctx context.Context, rule *models.EscalationRule) error {
	return s.EscalationRuleRepo.Create(ctx, rule)
}

// GetRule retrieves an escalation rule by ID
func (s *EscalationServiceImpl) GetRule(ctx context.Context, id string) (*models.EscalationRule, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid rule ID")
	}

	return s.EscalationRuleRepo.FindByID(ctx, objID)
}

// ListRules retrieves all escalation rules
func (s *EscalationServiceImpl) ListRules(ctx context.Context) ([]models.EscalationRule, error) {
	return s.EscalationRuleRepo.FindAll(ctx)
}

// UpdateRule updates an escalation rule
func (s *EscalationServiceImpl) UpdateRule(ctx context.Context, id string, updates map[string]interface{}) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid rule ID")
	}

	bsonUpdates := bson.M{}
	for k, v := range updates {
		bsonUpdates[k] = v
	}

	return s.EscalationRuleRepo.Update(ctx, objID, bsonUpdates)
}

// DeleteRule deletes an escalation rule
func (s *EscalationServiceImpl) DeleteRule(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid rule ID")
	}

	return s.EscalationRuleRepo.Delete(ctx, objID)
}
