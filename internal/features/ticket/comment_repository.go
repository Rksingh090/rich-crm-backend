package ticket

import (
	"context"
	"errors"
	"time"

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
	collection *mongo.Collection
}

// NewTicketCommentRepository creates a new ticket comment repository
func NewTicketCommentRepository(db *database.MongodbDB) TicketCommentRepository {
	return &TicketCommentRepositoryImpl{
		collection: db.DB.Collection("ticket_comments"),
	}
}

// Create inserts a new comment
func (r *TicketCommentRepositoryImpl) Create(ctx context.Context, comment *TicketComment) error {
	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, comment)
	if err != nil {
		return err
	}

	comment.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByTicketID retrieves all comments for a ticket
func (r *TicketCommentRepositoryImpl) FindByTicketID(ctx context.Context, ticketID primitive.ObjectID) ([]TicketComment, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := r.collection.Find(ctx, bson.M{"ticket_id": ticketID}, opts)
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
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("comment not found")
	}

	return nil
}
