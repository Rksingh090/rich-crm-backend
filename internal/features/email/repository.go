package email

import (
	"context"
	"fmt"
	"time"

	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EmailRepository struct {
	db *database.MongodbDB
}

func NewEmailRepository(db *database.MongodbDB) *EmailRepository {
	return &EmailRepository{
		db: db,
	}
}

func (r *EmailRepository) Create(ctx context.Context, email *Email) error {
	email.CreatedAt = time.Now()
	collection := r.db.GetTenantDB(email.TenantID.Hex()).Collection("emails")
	_, err := collection.InsertOne(ctx, email)
	return err
}

func (r *EmailRepository) UpdateStatus(
	ctx context.Context,
	id primitive.ObjectID,
	status EmailStatus,
	errorMsg string,
) error {
	// Attempt to get tenant from context if possible, or we might need it passed in.
	// For now, if we don't have it, this might be tricky.
	// However, usually it should be in the context.
	var tenantID string
	if val, ok := ctx.Value("tenant_id").(string); ok {
		tenantID = val
	} else if val, ok := ctx.Value("tenant_id").(primitive.ObjectID); ok {
		tenantID = val.Hex()
	}

	if tenantID == "" {
		// Fallback to searching all (but we shouldn't really have to)
		// and we don't have a good way to know which tenant this email belongs to without it being in context.
		return fmt.Errorf("tenant_id missing in context")
	}

	update := bson.M{
		"$set": bson.M{
			"status":       status,
			"errorMessage": errorMsg,
		},
	}
	if status == EmailSent {
		update["$set"].(bson.M)["sentAt"] = time.Now()
	}

	collection := r.db.GetTenantDB(tenantID).Collection("emails")
	_, err := collection.UpdateByID(ctx, id, update)
	return err
}
