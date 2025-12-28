package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-crm/internal/database"
	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TicketRepository defines the interface for ticket data operations
type TicketRepository interface {
	Create(ctx context.Context, ticket *models.Ticket) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.Ticket, error)
	FindAll(ctx context.Context, filter bson.M, page, limit int64, sortBy string, sortOrder string) ([]models.Ticket, int64, error)
	Update(ctx context.Context, id primitive.ObjectID, updates bson.M) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	FindByCustomer(ctx context.Context, customerID primitive.ObjectID, page, limit int64) ([]models.Ticket, int64, error)
	FindByAssignee(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]models.Ticket, int64, error)
	FindOverdueSLA(ctx context.Context) ([]models.Ticket, error)
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.TicketStatus, historyEntry models.StatusHistoryEntry) error
	GetNextTicketNumber(ctx context.Context) (string, error)
}

// TicketRepositoryImpl implements TicketRepository
type TicketRepositoryImpl struct {
	collection *mongo.Collection
}

// NewTicketRepository creates a new ticket repository
func NewTicketRepository(db *database.MongodbDB) TicketRepository {
	return &TicketRepositoryImpl{
		collection: db.DB.Collection("tickets"),
	}
}

// Create inserts a new ticket
func (r *TicketRepositoryImpl) Create(ctx context.Context, ticket *models.Ticket) error {
	ticket.CreatedAt = time.Now()
	ticket.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, ticket)
	if err != nil {
		return err
	}

	ticket.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByID retrieves a ticket by ID
func (r *TicketRepositoryImpl) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Ticket, error) {
	var ticket models.Ticket
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&ticket)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("ticket not found")
		}
		return nil, err
	}
	return &ticket, nil
}

// FindAll retrieves tickets with filtering, pagination, and sorting
func (r *TicketRepositoryImpl) FindAll(ctx context.Context, filter bson.M, page, limit int64, sortBy string, sortOrder string) ([]models.Ticket, int64, error) {
	// Count total documents
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Calculate skip
	skip := (page - 1) * limit

	// Determine sort order
	sortValue := 1
	if sortOrder == "desc" {
		sortValue = -1
	}

	// Find options
	findOptions := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.D{{Key: sortBy, Value: sortValue}})

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var tickets []models.Ticket
	if err = cursor.All(ctx, &tickets); err != nil {
		return nil, 0, err
	}

	return tickets, total, nil
}

// Update updates a ticket
func (r *TicketRepositoryImpl) Update(ctx context.Context, id primitive.ObjectID, updates bson.M) error {
	updates["updated_at"] = time.Now()

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("ticket not found")
	}

	return nil
}

// Delete removes a ticket
func (r *TicketRepositoryImpl) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("ticket not found")
	}

	return nil
}

// FindByCustomer retrieves tickets for a specific customer
func (r *TicketRepositoryImpl) FindByCustomer(ctx context.Context, customerID primitive.ObjectID, page, limit int64) ([]models.Ticket, int64, error) {
	filter := bson.M{"customer_id": customerID}
	return r.FindAll(ctx, filter, page, limit, "created_at", "desc")
}

// FindByAssignee retrieves tickets assigned to a specific user
func (r *TicketRepositoryImpl) FindByAssignee(ctx context.Context, userID primitive.ObjectID, page, limit int64) ([]models.Ticket, int64, error) {
	filter := bson.M{"assigned_to": userID}
	return r.FindAll(ctx, filter, page, limit, "created_at", "desc")
}

// FindOverdueSLA finds tickets that have breached their SLA
func (r *TicketRepositoryImpl) FindOverdueSLA(ctx context.Context) ([]models.Ticket, error) {
	now := time.Now()
	filter := bson.M{
		"$or": []bson.M{
			{
				"response_due_date": bson.M{"$lt": now},
				"first_response_at": nil,
			},
			{
				"due_date": bson.M{"$lt": now},
				"status":   bson.M{"$nin": []models.TicketStatus{models.TicketStatusResolved, models.TicketStatusClosed}},
			},
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tickets []models.Ticket
	if err = cursor.All(ctx, &tickets); err != nil {
		return nil, err
	}

	return tickets, nil
}

// UpdateStatus updates the ticket status and adds to history
func (r *TicketRepositoryImpl) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.TicketStatus, historyEntry models.StatusHistoryEntry) error {
	updates := bson.M{
		"status":     status,
		"updated_at": time.Now(),
	}

	// Add resolved/closed timestamps
	if status == models.TicketStatusResolved {
		updates["resolved_at"] = time.Now()
	} else if status == models.TicketStatusClosed {
		updates["closed_at"] = time.Now()
	}

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set":  updates,
			"$push": bson.M{"status_history": historyEntry},
		},
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("ticket not found")
	}

	return nil
}

// GetNextTicketNumber generates the next ticket number
func (r *TicketRepositoryImpl) GetNextTicketNumber(ctx context.Context) (string, error) {
	// Find the latest ticket
	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})
	var lastTicket models.Ticket
	err := r.collection.FindOne(ctx, bson.M{}, opts).Decode(&lastTicket)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// First ticket
			return "TKT-000001", nil
		}
		return "", err
	}

	// Extract number from last ticket and increment
	var lastNumber int
	_, err = fmt.Sscanf(lastTicket.TicketNumber, "TKT-%d", &lastNumber)
	if err != nil {
		// Fallback if parsing fails
		return "TKT-000001", nil
	}

	nextNumber := lastNumber + 1
	return fmt.Sprintf("TKT-%06d", nextNumber), nil
}
