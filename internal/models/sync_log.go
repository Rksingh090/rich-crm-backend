package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SyncLog struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	SyncSettingID  primitive.ObjectID `json:"sync_setting_id" bson:"sync_setting_id"`
	StartTime      time.Time          `json:"start_time" bson:"start_time"`
	EndTime        time.Time          `json:"end_time" bson:"end_time"`
	Status         string             `json:"status" bson:"status"` // "success", "failed", "in_progress"
	ProcessedCount int                `json:"processed_count" bson:"processed_count"`
	Error          string             `json:"error,omitempty" bson:"error,omitempty"`
}
