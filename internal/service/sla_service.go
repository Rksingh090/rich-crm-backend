package service

import (
	"context"
	"errors"
	"time"

	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SLAStatus represents the current status of an SLA
type SLAStatus struct {
	Status                  string     `json:"status"`                              // "on_time", "at_risk", "breached", "no_sla"
	ResponseTimeRemaining   *int       `json:"response_time_remaining,omitempty"`   // minutes
	ResolutionTimeRemaining *int       `json:"resolution_time_remaining,omitempty"` // minutes
	ResponseDueDate         *time.Time `json:"response_due_date,omitempty"`
	ResolutionDueDate       *time.Time `json:"resolution_due_date,omitempty"`
	IsResponseBreached      bool       `json:"is_response_breached"`
	IsResolutionBreached    bool       `json:"is_resolution_breached"`
}

// SLAMetrics represents overall SLA performance metrics
type SLAMetrics struct {
	TotalTickets      int     `json:"total_tickets"`
	SLAMet            int     `json:"sla_met"`
	SLABreached       int     `json:"sla_breached"`
	ComplianceRate    float64 `json:"compliance_rate"`
	AvgResponseTime   float64 `json:"avg_response_time"`   // minutes
	AvgResolutionTime float64 `json:"avg_resolution_time"` // minutes
}

// SLATrend represents SLA performance over time
type SLATrend struct {
	Date     string `json:"date"`
	Met      int    `json:"met"`
	Breached int    `json:"breached"`
	Total    int    `json:"total"`
}

// SLAViolation represents an active SLA violation
type SLAViolation struct {
	TicketID     string                `json:"ticket_id"`
	TicketNumber string                `json:"ticket_number"`
	Subject      string                `json:"subject"`
	Priority     models.TicketPriority `json:"priority"`
	BreachType   string                `json:"breach_type"` // "response" or "resolution"
	BreachedAt   time.Time             `json:"breached_at"`
	TimeOverdue  int                   `json:"time_overdue"` // minutes
}

// SLAPervice defines the interface for SLA policy management
type SLAPervice interface { // Typo in original interface name? No SLAService
	CreatePolicy(ctx context.Context, policy *models.SLAPolicy) error
	GetPolicy(ctx context.Context, id string) (*models.SLAPolicy, error)
	ListPolicies(ctx context.Context) ([]models.SLAPolicy, error)
	UpdatePolicy(ctx context.Context, id string, updates map[string]interface{}) error
	DeletePolicy(ctx context.Context, id string) error
}

// SLAService defines the interface for SLA policy management and metrics
type SLAService interface {
	CreatePolicy(ctx context.Context, policy *models.SLAPolicy) error
	GetPolicy(ctx context.Context, id string) (*models.SLAPolicy, error)
	ListPolicies(ctx context.Context) ([]models.SLAPolicy, error)
	UpdatePolicy(ctx context.Context, id string, updates map[string]interface{}) error
	DeletePolicy(ctx context.Context, id string) error

	// Metrics and tracking
	CalculateSLAStatus(ctx context.Context, ticket *models.Ticket, policy *models.SLAPolicy) *SLAStatus
	GetSLAMetrics(ctx context.Context, startDate, endDate time.Time) (*SLAMetrics, error)
	GetSLAViolations(ctx context.Context) ([]SLAViolation, error)
	GetSLATrends(ctx context.Context, days int) ([]SLATrend, error)
}

// SLAServiceImpl implements SLAService
type SLAServiceImpl struct {
	SLAPolicyRepo repository.SLAPolicyRepository
	TicketRepo    repository.TicketRepository
}

// NewSLAService creates a new SLA service
func NewSLAService(slaPolicyRepo repository.SLAPolicyRepository, ticketRepo repository.TicketRepository) SLAService {
	return &SLAServiceImpl{
		SLAPolicyRepo: slaPolicyRepo,
		TicketRepo:    ticketRepo,
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

// CalculateSLAStatus calculates the current SLA status for a ticket
func (s *SLAServiceImpl) CalculateSLAStatus(ctx context.Context, ticket *models.Ticket, policy *models.SLAPolicy) *SLAStatus {
	if policy == nil {
		return &SLAStatus{
			Status:               "no_sla",
			IsResponseBreached:   false,
			IsResolutionBreached: false,
		}
	}

	now := time.Now()
	status := &SLAStatus{}

	// Calculate response SLA
	if ticket.ResponseDueDate != nil {
		status.ResponseDueDate = ticket.ResponseDueDate

		if ticket.FirstResponseAt == nil {
			// No response yet
			timeRemaining := int(ticket.ResponseDueDate.Sub(now).Minutes())
			status.ResponseTimeRemaining = &timeRemaining

			if now.After(*ticket.ResponseDueDate) {
				status.IsResponseBreached = true
			}
		} else {
			// Response provided
			if ticket.FirstResponseAt.After(*ticket.ResponseDueDate) {
				status.IsResponseBreached = true
			}
		}
	}

	// Calculate resolution SLA
	if ticket.DueDate != nil {
		status.ResolutionDueDate = ticket.DueDate

		if ticket.ResolvedAt == nil && ticket.ClosedAt == nil {
			// Not resolved yet
			timeRemaining := int(ticket.DueDate.Sub(now).Minutes())
			status.ResolutionTimeRemaining = &timeRemaining

			if now.After(*ticket.DueDate) {
				status.IsResolutionBreached = true
			}
		} else {
			// Ticket resolved/closed
			resolvedTime := ticket.ResolvedAt
			if resolvedTime == nil {
				resolvedTime = ticket.ClosedAt
			}

			if resolvedTime.After(*ticket.DueDate) {
				status.IsResolutionBreached = true
			}
		}
	}

	// Determine overall status
	if status.IsResponseBreached || status.IsResolutionBreached {
		status.Status = "breached"
	} else {
		// Check if at risk (less than 25% time remaining)
		atRisk := false

		if status.ResponseTimeRemaining != nil && *status.ResponseTimeRemaining > 0 {
			threshold := float64(policy.ResponseTime) * 0.25
			if float64(*status.ResponseTimeRemaining) < threshold {
				atRisk = true
			}
		}

		if status.ResolutionTimeRemaining != nil && *status.ResolutionTimeRemaining > 0 {
			threshold := float64(policy.ResolutionTime) * 0.25
			if float64(*status.ResolutionTimeRemaining) < threshold {
				atRisk = true
			}
		}

		if atRisk {
			status.Status = "at_risk"
		} else {
			status.Status = "on_time"
		}
	}

	return status
}

// GetSLAMetrics retrieves overall SLA performance metrics
func (s *SLAServiceImpl) GetSLAMetrics(ctx context.Context, startDate, endDate time.Time) (*SLAMetrics, error) {
	// Get all tickets in date range
	tickets, _, err := s.TicketRepo.FindAll(ctx, bson.M{
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}, 1, 10000, "created_at", "desc")
	if err != nil {
		return nil, err
	}

	metrics := &SLAMetrics{
		TotalTickets: len(tickets),
	}

	var totalResponseTime, totalResolutionTime float64
	var responseCount, resolutionCount int

	for _, ticket := range tickets {
		if ticket.SLAPolicyID == nil {
			continue
		}

		// Check response SLA
		if ticket.ResponseDueDate != nil {
			if ticket.FirstResponseAt != nil {
				responseTime := ticket.FirstResponseAt.Sub(ticket.CreatedAt).Minutes()
				totalResponseTime += responseTime
				responseCount++

				if ticket.FirstResponseAt.After(*ticket.ResponseDueDate) {
					metrics.SLABreached++
				} else {
					metrics.SLAMet++
				}
			} else if time.Now().After(*ticket.ResponseDueDate) {
				metrics.SLABreached++
			}
		}

		// Check resolution SLA
		if ticket.DueDate != nil {
			if ticket.ResolvedAt != nil || ticket.ClosedAt != nil {
				resolvedTime := ticket.ResolvedAt
				if resolvedTime == nil {
					resolvedTime = ticket.ClosedAt
				}

				resolutionTime := resolvedTime.Sub(ticket.CreatedAt).Minutes()
				totalResolutionTime += resolutionTime
				resolutionCount++

				if resolvedTime.After(*ticket.DueDate) {
					metrics.SLABreached++
				} else {
					metrics.SLAMet++
				}
			} else if time.Now().After(*ticket.DueDate) {
				metrics.SLABreached++
			}
		}
	}

	// Calculate averages
	if responseCount > 0 {
		metrics.AvgResponseTime = totalResponseTime / float64(responseCount)
	}
	if resolutionCount > 0 {
		metrics.AvgResolutionTime = totalResolutionTime / float64(resolutionCount)
	}

	// Calculate compliance rate
	total := metrics.SLAMet + metrics.SLABreached
	if total > 0 {
		metrics.ComplianceRate = float64(metrics.SLAMet) / float64(total) * 100
	}

	return metrics, nil
}

// GetSLAViolations retrieves all active SLA violations
func (s *SLAServiceImpl) GetSLAViolations(ctx context.Context) ([]SLAViolation, error) {
	now := time.Now()

	// Find all open tickets with SLA policies
	tickets, _, err := s.TicketRepo.FindAll(ctx, bson.M{
		"status": bson.M{
			"$nin": []string{"resolved", "closed"},
		},
		"sla_policy_id": bson.M{"$ne": nil},
	}, 1, 10000, "created_at", "desc")
	if err != nil {
		return nil, err
	}

	var violations []SLAViolation

	for _, ticket := range tickets {
		// Check response SLA breach
		if ticket.ResponseDueDate != nil && ticket.FirstResponseAt == nil {
			if now.After(*ticket.ResponseDueDate) {
				overdue := int(now.Sub(*ticket.ResponseDueDate).Minutes())
				violations = append(violations, SLAViolation{
					TicketID:     ticket.ID.Hex(),
					TicketNumber: ticket.TicketNumber,
					Subject:      ticket.Subject,
					Priority:     ticket.Priority,
					BreachType:   "response",
					BreachedAt:   *ticket.ResponseDueDate,
					TimeOverdue:  overdue,
				})
			}
		}

		// Check resolution SLA breach
		if ticket.DueDate != nil && ticket.ResolvedAt == nil && ticket.ClosedAt == nil {
			if now.After(*ticket.DueDate) {
				overdue := int(now.Sub(*ticket.DueDate).Minutes())
				violations = append(violations, SLAViolation{
					TicketID:     ticket.ID.Hex(),
					TicketNumber: ticket.TicketNumber,
					Subject:      ticket.Subject,
					Priority:     ticket.Priority,
					BreachType:   "resolution",
					BreachedAt:   *ticket.DueDate,
					TimeOverdue:  overdue,
				})
			}
		}
	}

	return violations, nil
}

// GetSLATrends retrieves SLA performance trends
func (s *SLAServiceImpl) GetSLATrends(ctx context.Context, days int) ([]SLATrend, error) {
	trends := make([]SLATrend, days)

	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		startOfDay := time.Now().AddDate(0, 0, -i).Truncate(24 * time.Hour)
		endOfDay := startOfDay.Add(24 * time.Hour)

		// Get tickets created on this day
		tickets, _, err := s.TicketRepo.FindAll(ctx, bson.M{
			"created_at": bson.M{
				"$gte": startOfDay,
				"$lt":  endOfDay,
			},
			"sla_policy_id": bson.M{"$ne": nil},
		}, 1, 10000, "created_at", "desc")
		if err != nil {
			return nil, err
		}

		trend := SLATrend{
			Date:  date,
			Total: len(tickets),
		}

		for _, ticket := range tickets {
			// Simplified check - in reality, you'd check both response and resolution
			if ticket.DueDate != nil {
				if ticket.ResolvedAt != nil || ticket.ClosedAt != nil {
					resolvedTime := ticket.ResolvedAt
					if resolvedTime == nil {
						resolvedTime = ticket.ClosedAt
					}

					if resolvedTime.After(*ticket.DueDate) {
						trend.Breached++
					} else {
						trend.Met++
					}
				}
			}
		}

		trends[days-1-i] = trend
	}

	return trends, nil
}
