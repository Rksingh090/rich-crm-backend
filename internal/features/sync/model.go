package sync

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ModuleSyncConfig struct {
	ModuleName  string            `json:"module_name" bson:"module_name"`
	Mapping     map[string]string `json:"mapping" bson:"mapping"` // CRM Field -> Target DB Column
	SyncDeletes bool              `json:"sync_deletes" bson:"sync_deletes"`
}

type SyncSetting struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name           string             `json:"name" bson:"name"`
	Modules        []ModuleSyncConfig `json:"modules" bson:"modules"`
	TargetDBType   string             `json:"target_db_type" bson:"target_db_type"` // "postgres", "mysql", "sqlserver", "mongodb"
	TargetDBConfig map[string]string  `json:"target_db_config" bson:"target_db_config"`
	LastSyncAt     time.Time          `json:"last_sync_at" bson:"last_sync_at"`
	IsActive       bool               `json:"is_active" bson:"is_active"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" bson:"updated_at"`
}

type SyncLog struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	SyncSettingID  primitive.ObjectID `json:"sync_setting_id" bson:"sync_setting_id"`
	StartTime      time.Time          `json:"start_time" bson:"start_time"`
	EndTime        time.Time          `json:"end_time" bson:"end_time"`
	Status         string             `json:"status" bson:"status"` // "success", "failed", "in_progress"
	ProcessedCount int                `json:"processed_count" bson:"processed_count"`
	Error          string             `json:"error,omitempty" bson:"error,omitempty"`
}
