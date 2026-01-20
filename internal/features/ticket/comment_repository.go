package ticket

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-crm/internal/common/models"
	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TicketCommentRepository defines the interface for ticket comment operations
type TicketCommentRepository interface {
	Create(ctx context.Context, comment *TicketComment) error
	FindByTicketID(ctx context.Context, ticketID primitive.ObjectID) ([]TicketComment, error)
	Delete(ctx context.Context, id primitive.ObjectID) error
}

// TicketCommentRepositoryImpl implements TicketCommentRepository
type TicketCommentRepositoryImpl struct {
	db *database.MongodbDB
}

// NewTicketCommentRepository creates a new ticket comment repository
func NewTicketCommentRepository(db *database.MongodbDB) TicketCommentRepository {
	return &TicketCommentRepositoryImpl{
		db: db,
	}
}

func (r *TicketCommentRepositoryImpl) getCollection(ctx context.Context) (*mongo.Collection, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant context missing")
	}
	return r.db.GetTenantDB(tenantID).Collection("ticket_comments"), nil
}

// Create inserts a new comment
func (r *TicketCommentRepositoryImpl) Create(ctx context.Context, comment *TicketComment) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()

	// Ensure tenant ID is set if available
	if tenantID, ok := ctx.Value(models.TenantIDKey).(string); ok && tenantID != "" {
		comment.TenantID, _ = primitive.ObjectIDFromHex(tenantID)
	}

	result, err := coll.InsertOne(ctx, comment)
	if err != nil {
		return err
	}

	comment.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByTicketID retrieves all comments for a ticket
func (r *TicketCommentRepositoryImpl) FindByTicketID(ctx context.Context, ticketID primitive.ObjectID) ([]TicketComment, error) {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := coll.Find(ctx, bson.M{"ticket_id": ticketID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var comments []TicketComment
	if err = cursor.All(ctx, &comments); err != nil {
		return nil, err
	}

	return comments, nil
}

// Delete removes a comment
func (r *TicketCommentRepositoryImpl) Delete(ctx context.Context, id primitive.ObjectID) error {
	coll, err := r.getCollection(ctx)
	if err != nil {
		return err
	}

	result, err := coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("comment not found")
	}

	return nil
}
