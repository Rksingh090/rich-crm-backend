package ticket

import (
	"context"
	"errors"
	"fmt"
	"time"

	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/notification"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TicketService defines the interface for ticket business logic
type TicketService interface {
	CreateTicket(ctx context.Context, ticket *Ticket, createdBy primitive.ObjectID) error
	GetTicket(ctx context.Context, id string) (*Ticket, error)
	ListTickets(ctx context.Context, filters map[string]interface{}, page, limit int64, sortBy, sortOrder string) ([]Ticket, int64, error)
	UpdateTicket(ctx context.Context, id string, updates map[string]interface{}, updatedBy primitive.ObjectID) error
	DeleteTicket(ctx context.Context, id string, deletedBy primitive.ObjectID) error

	// Status Management
	UpdateStatus(ctx context.Context, id string, status TicketStatus, comment string, changedBy primitive.ObjectID) error
	GetStatusHistory(ctx context.Context, id string) ([]StatusHistoryEntry, error)

	// Assignment
	AssignTicket(ctx context.Context, id string, assignedTo primitive.ObjectID, assignedBy primitive.ObjectID) error
	UnassignTicket(ctx context.Context, id string, unassignedBy primitive.ObjectID) error
	GetMyTickets(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]Ticket, int64, error)
	GetCustomerTickets(ctx context.Context, customerID primitive.ObjectID, page, limit int64) ([]Ticket, int64, error)

	// Comments
	AddComment(ctx context.Context, ticketID string, comment *TicketComment) error
	ListComments(ctx context.Context, ticketID string) ([]TicketComment, error)

	// SLA Management
	CalculateDueDates(ctx context.Context, ticket *Ticket) error
	CheckSLABreach(ctx context.Context, ticketID string) (bool, error)
	GetOverdueSLATickets(ctx context.Context) ([]Ticket, error)

	// Multi-Channel
	CreateTicketFromEmail(ctx context.Context, subject, description, customerEmail, customerName string, metadata map[string]interface{}) error
	CreateTicketFromChat(ctx context.Context, subject, description, customerEmail, customerName string, metadata map[string]interface{}) error
	CreateTicketFromPortal(ctx context.Context, ticket *Ticket, createdBy primitive.ObjectID) error
}

// TicketServiceImpl implements TicketService
type TicketServiceImpl struct {
	TicketRepo          TicketRepository
	SLAPolicyRepo       SLAPolicyRepository
	CommentRepo         TicketCommentRepository
	AuditService        audit.AuditService
	NotificationService notification.NotificationService
}

// NewTicketService creates a new ticket service
func NewTicketService(
	ticketRepo TicketRepository,
	slaPolicyRepo SLAPolicyRepository,
	commentRepo TicketCommentRepository,
	auditService audit.AuditService,
	notificationService notification.NotificationService,
) TicketService {
	return &TicketServiceImpl{
		TicketRepo:          ticketRepo,
		SLAPolicyRepo:       slaPolicyRepo,
		CommentRepo:         commentRepo,
		AuditService:        auditService,
		NotificationService: notificationService,
	}
}

// CreateTicket creates a new ticket
func (s *TicketServiceImpl) CreateTicket(ctx context.Context, t *Ticket, createdBy primitive.ObjectID) error {
	// Generate ticket number
	ticketNumber, err := s.TicketRepo.GetNextTicketNumber(ctx)
	if err != nil {
		return err
	}
	t.TicketNumber = ticketNumber

	// Set initial status
	if t.Status == "" {
		t.Status = TicketStatusNew
	}

	// Initialize status history
	t.StatusHistory = []StatusHistoryEntry{
		{
			Status:    t.Status,
			ChangedBy: createdBy,
			ChangedAt: time.Now(),
			Comment:   "Ticket created",
		},
	}

	// Calculate SLA due dates
	if err := s.CalculateDueDates(ctx, t); err != nil {
		return err
	}

	// Create ticket
	if err := s.TicketRepo.Create(ctx, t); err != nil {
		return err
	}

	// Audit log
	changes := map[string]common_models.Change{
		"ticket_number": {Old: nil, New: t.TicketNumber},
		"subject":       {Old: nil, New: t.Subject},
		"priority":      {Old: nil, New: t.Priority},
		"status":        {Old: nil, New: t.Status},
	}
	_ = s.AuditService.LogChange(ctx, common_models.AuditActionCreate, "tickets", t.ID.Hex(), changes)

	return nil
}

// GetTicket retrieves a ticket by ID
func (s *TicketServiceImpl) GetTicket(ctx context.Context, id string) (*Ticket, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid ticket ID")
	}

	return s.TicketRepo.FindByID(ctx, objID)
}

// ListTickets retrieves tickets with filtering and pagination
func (s *TicketServiceImpl) ListTickets(ctx context.Context, filters map[string]interface{}, page, limit int64, sortBy, sortOrder string) ([]Ticket, int64, error) {
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
	changes := make(map[string]common_models.Change)
	if subject, ok := updates["subject"]; ok {
		changes["subject"] = common_models.Change{Old: oldTicket.Subject, New: subject}
	}
	if description, ok := updates["description"]; ok {
		changes["description"] = common_models.Change{Old: oldTicket.Description, New: description}
	}
	if priority, ok := updates["priority"]; ok {
		changes["priority"] = common_models.Change{Old: oldTicket.Priority, New: priority}
	}

	if len(changes) > 0 {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionUpdate, "tickets", objID.Hex(), changes)
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
	t, err := s.TicketRepo.FindByID(ctx, objID)
	if err != nil {
		return err
	}

	// Delete ticket
	if err := s.TicketRepo.Delete(ctx, objID); err != nil {
		return err
	}

	// Audit log
	changes := map[string]common_models.Change{
		"ticket_number": {Old: t.TicketNumber, New: nil},
		"subject":       {Old: t.Subject, New: nil},
	}
	_ = s.AuditService.LogChange(ctx, common_models.AuditActionDelete, "tickets", objID.Hex(), changes)

	return nil
}

// UpdateStatus updates the ticket status
func (s *TicketServiceImpl) UpdateStatus(ctx context.Context, id string, status TicketStatus, comment string, changedBy primitive.ObjectID) error {
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
	validStatuses := map[TicketStatus]bool{
		TicketStatusNew:      true,
		TicketStatusOpen:     true,
		TicketStatusPending:  true,
		TicketStatusResolved: true,
		TicketStatusClosed:   true,
	}

	if !validStatuses[status] {
		return errors.New("invalid status")
	}

	// Create history entry
	historyEntry := StatusHistoryEntry{
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
	changes := map[string]common_models.Change{
		"status": {Old: oldTicket.Status, New: status},
	}
	_ = s.AuditService.LogChange(ctx, common_models.AuditActionUpdate, "tickets", objID.Hex(), changes)

	return nil
}

// GetStatusHistory retrieves the status history of a ticket
func (s *TicketServiceImpl) GetStatusHistory(ctx context.Context, id string) ([]StatusHistoryEntry, error) {
	t, err := s.GetTicket(ctx, id)
	if err != nil {
		return nil, err
	}

	return t.StatusHistory, nil
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
	changes := map[string]common_models.Change{
		"assigned_to": {Old: oldAssignee, New: assignedTo.Hex()},
	}
	_ = s.AuditService.LogChange(ctx, common_models.AuditActionUpdate, "tickets", objID.Hex(), changes)

	// Send notification to assignee
	_ = s.NotificationService.CreateNotification(ctx, assignedTo, "Ticket Assigned", fmt.Sprintf("You have been assigned ticket %s: %s", oldTicket.TicketNumber, oldTicket.Subject), notification.NotificationTypeTask, fmt.Sprintf("/dashboard/modules/tickets/%s", id))

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
	changes := map[string]common_models.Change{
		"assigned_to": {Old: oldAssignee, New: nil},
	}
	_ = s.AuditService.LogChange(ctx, common_models.AuditActionUpdate, "tickets", objID.Hex(), changes)

	return nil
}

// GetMyTickets retrieves tickets assigned to a user
func (s *TicketServiceImpl) GetMyTickets(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]Ticket, int64, error) {
	return s.TicketRepo.FindByAssignee(ctx, userID, page, limit)
}

// GetCustomerTickets retrieves tickets for a customer
func (s *TicketServiceImpl) GetCustomerTickets(ctx context.Context, customerID primitive.ObjectID, page, limit int64) ([]Ticket, int64, error) {
	return s.TicketRepo.FindByCustomer(ctx, customerID, page, limit)
}

// AddComment adds a comment to a ticket
func (s *TicketServiceImpl) AddComment(ctx context.Context, ticketID string, tComment *TicketComment) error {
	objID, err := primitive.ObjectIDFromHex(ticketID)
	if err != nil {
		return errors.New("invalid ticket ID")
	}

	// Verify ticket exists
	_, err = s.TicketRepo.FindByID(ctx, objID)
	if err != nil {
		return err
	}

	tComment.TicketID = objID

	if err := s.CommentRepo.Create(ctx, tComment); err != nil {
		return err
	}

	// Update first response time if this is the first response
	t, _ := s.TicketRepo.FindByID(ctx, objID)
	if t != nil && t.FirstResponseAt == nil && !tComment.IsInternal {
		now := time.Now()
		_ = s.TicketRepo.Update(ctx, objID, bson.M{"first_response_at": now})
	}

	// Audit log
	changes := map[string]common_models.Change{
		"comment_added": {Old: nil, New: tComment.ID.Hex()},
	}
	_ = s.AuditService.LogChange(ctx, common_models.AuditActionUpdate, "tickets", objID.Hex(), changes)

	return nil
}

// ListComments retrieves all comments for a ticket
func (s *TicketServiceImpl) ListComments(ctx context.Context, ticketID string) ([]TicketComment, error) {
	objID, err := primitive.ObjectIDFromHex(ticketID)
	if err != nil {
		return nil, errors.New("invalid ticket ID")
	}

	return s.CommentRepo.FindByTicketID(ctx, objID)
}

// CalculateDueDates calculates SLA due dates for a ticket
func (s *TicketServiceImpl) CalculateDueDates(ctx context.Context, t *Ticket) error {
	// Find SLA policy for the ticket's priority
	policy, err := s.SLAPolicyRepo.FindByPriority(ctx, t.Priority)
	if err != nil {
		return err
	}

	if policy == nil {
		// No SLA policy found, skip
		return nil
	}

	t.SLAPolicyID = &policy.ID

	now := time.Now()

	// Calculate response due date
	responseDue := now.Add(time.Duration(policy.ResponseTime) * time.Minute)
	t.ResponseDueDate = &responseDue

	// Calculate resolution due date
	resolutionDue := now.Add(time.Duration(policy.ResolutionTime) * time.Minute)
	t.DueDate = &resolutionDue

	return nil
}

// CheckSLABreach checks if a ticket has breached its SLA
func (s *TicketServiceImpl) CheckSLABreach(ctx context.Context, ticketID string) (bool, error) {
	t, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return false, err
	}

	now := time.Now()

	// Check response SLA breach
	if t.ResponseDueDate != nil && t.FirstResponseAt == nil {
		if now.After(*t.ResponseDueDate) {
			return true, nil
		}
	}

	// Check resolution SLA breach
	if t.DueDate != nil {
		if t.Status != TicketStatusResolved && t.Status != TicketStatusClosed {
			if now.After(*t.DueDate) {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetOverdueSLATickets retrieves all tickets with SLA breaches
func (s *TicketServiceImpl) GetOverdueSLATickets(ctx context.Context) ([]Ticket, error) {
	return s.TicketRepo.FindOverdueSLA(ctx)
}

// CreateTicketFromEmail creates a ticket from an email
func (s *TicketServiceImpl) CreateTicketFromEmail(ctx context.Context, subject, description, customerEmail, customerName string, metadata map[string]interface{}) error {
	t := &Ticket{
		Subject:         subject,
		Description:     description,
		Channel:         TicketChannelEmail,
		ChannelMetadata: metadata,
		CustomerEmail:   customerEmail,
		CustomerName:    customerName,
		Priority:        TicketPriorityMedium, // Default priority
		Status:          TicketStatusNew,
	}

	// Use system user ID for automated ticket creation
	systemUserID := primitive.NewObjectID() // In production, use actual system user ID

	return s.CreateTicket(ctx, t, systemUserID)
}

// CreateTicketFromChat creates a ticket from a chat conversation
func (s *TicketServiceImpl) CreateTicketFromChat(ctx context.Context, subject, description, customerEmail, customerName string, metadata map[string]interface{}) error {
	t := &Ticket{
		Subject:         subject,
		Description:     description,
		Channel:         TicketChannelChat,
		ChannelMetadata: metadata,
		CustomerEmail:   customerEmail,
		CustomerName:    customerName,
		Priority:        TicketPriorityMedium,
		Status:          TicketStatusNew,
	}

	systemUserID := primitive.NewObjectID()

	return s.CreateTicket(ctx, t, systemUserID)
}

// CreateTicketFromPortal creates a ticket from the customer portal
func (s *TicketServiceImpl) CreateTicketFromPortal(ctx context.Context, t *Ticket, createdBy primitive.ObjectID) error {
	t.Channel = TicketChannelPortal
	t.CustomerID = &createdBy

	return s.CreateTicket(ctx, t, createdBy)
}
