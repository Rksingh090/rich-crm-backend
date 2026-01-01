package emails

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EmailStatus string

const (
	EmailQueued EmailStatus = "queued"
	EmailSent   EmailStatus = "sent"
	EmailFailed EmailStatus = "failed"
)

type Email struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrgID      primitive.ObjectID `bson:"orgId" json:"orgId"`
	From       string             `bson:"from" json:"from"`
	To         []string           `bson:"to" json:"to"`
	Cc         []string           `bson:"cc,omitempty" json:"cc,omitempty"`
	Bcc        []string           `bson:"bcc,omitempty" json:"bcc,omitempty"`
	Subject    string             `bson:"subject" json:"subject"`
	HtmlBody   string             `bson:"htmlBody,omitempty" json:"htmlBody,omitempty"`
	TextBody   string             `bson:"textBody,omitempty" json:"textBody,omitempty"`
	Status     EmailStatus        `bson:"status" json:"status"`
	EntityType string             `bson:"entityType,omitempty" json:"entityType,omitempty"`
	EntityID   primitive.ObjectID `bson:"entityId,omitempty" json:"entityId,omitempty"`
	ErrorMsg   string             `bson:"errorMessage,omitempty" json:"errorMessage,omitempty"`
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
	SentAt     *time.Time         `bson:"sentAt,omitempty" json:"sentAt,omitempty"`
}
