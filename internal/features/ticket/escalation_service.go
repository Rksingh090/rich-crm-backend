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

// EscalationService defines the interface for escalation management
type EscalationService interface {
	ProcessEscalations(ctx context.Context) error
	EvaluateRules(ctx context.Context, ticket *Ticket) ([]EscalationRule, error)
	ExecuteEscalation(ctx context.Context, ticket *Ticket, rule *EscalationRule) error
	CreateRule(ctx context.Context, rule *EscalationRule) error
	GetRule(ctx context.Context, id string) (*EscalationRule, error)
	ListRules(ctx context.Context) ([]EscalationRule, error)
	UpdateRule(ctx context.Context, id string, updates map[string]interface{}) error
	DeleteRule(ctx context.Context, id string) error
}

// EscalationServiceImpl implements EscalationService
type EscalationServiceImpl struct {
	EscalationRuleRepo  EscalationRuleRepository
	TicketRepo          TicketRepository
	AuditService        audit.AuditService
	NotificationService notification.NotificationService
}

// NewEscalationService creates a new escalation service
func NewEscalationService(
	escalationRuleRepo EscalationRuleRepository,
	ticketRepo TicketRepository,
	auditService audit.AuditService,
	notificationService notification.NotificationService,
) EscalationService {
	return &EscalationServiceImpl{
		EscalationRuleRepo:  escalationRuleRepo,
		TicketRepo:          ticketRepo,
		AuditService:        auditService,
		NotificationService: notificationService,
	}
}

// ProcessEscalations processes all tickets for escalation
func (s *EscalationServiceImpl) ProcessEscalations(ctx context.Context) error {
	// Get all open tickets
	tickets, _, err := s.TicketRepo.FindAll(ctx, bson.M{
		"status": bson.M{"$nin": []TicketStatus{TicketStatusResolved, TicketStatusClosed}},
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
			_ = s.ExecuteEscalation(ctx, &ticket, &rule)
		}
	}

	return nil
}

// EvaluateRules evaluates which escalation rules apply to a ticket
func (s *EscalationServiceImpl) EvaluateRules(ctx context.Context, ticket *Ticket) ([]EscalationRule, error) {
	rules, err := s.EscalationRuleRepo.FindActive(ctx)
	if err != nil {
		return nil, err
	}

	var applicableRules []EscalationRule
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
func (s *EscalationServiceImpl) ExecuteEscalation(ctx context.Context, ticket *Ticket, rule *EscalationRule) error {
	// Create escalation history entry
	escalationEntry := EscalationHistoryEntry{
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
	_ = s.TicketRepo.Update(ctx, ticket.ID, bson.M{
		"$push": bson.M{"escalation_history": escalationEntry},
	})

	// Audit log
	changes := map[string]common_models.Change{
		"escalation_level": {Old: ticket.EscalationLevel, New: ticket.EscalationLevel + 1},
		"escalated_to":     {Old: nil, New: rule.EscalateTo.Hex()},
	}
	_ = s.AuditService.LogChange(ctx, common_models.AuditActionUpdate, "tickets", ticket.ID.Hex(), changes)

	// Send notifications to escalated_to user
	_ = s.NotificationService.CreateNotification(ctx, rule.EscalateTo, "Ticket Escalated", fmt.Sprintf("Ticket %s has been escalated to you due to rule: %s", ticket.TicketNumber, rule.Name), notification.NotificationTypeSLA, fmt.Sprintf("/dashboard/modules/tickets/%s", ticket.ID.Hex()))

	return nil
}

// CreateRule creates a new escalation rule
func (s *EscalationServiceImpl) CreateRule(ctx context.Context, rule *EscalationRule) error {
	return s.EscalationRuleRepo.Create(ctx, rule)
}

// GetRule retrieves an escalation rule by ID
func (s *EscalationServiceImpl) GetRule(ctx context.Context, id string) (*EscalationRule, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid rule ID")
	}

	return s.EscalationRuleRepo.FindByID(ctx, objID)
}

// ListRules retrieves all escalation rules
func (s *EscalationServiceImpl) ListRules(ctx context.Context) ([]EscalationRule, error) {
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
