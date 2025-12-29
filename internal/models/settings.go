package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SettingsType string

const (
	SettingsTypeEmail       SettingsType = "email"
	SettingsTypeGeneral     SettingsType = "general"
	SettingsTypeFileSharing SettingsType = "file_sharing"
)

type EmailConfig struct {
	SMTPHost     string `json:"smtp_host" bson:"smtp_host"`
	SMTPPort     int    `json:"smtp_port" bson:"smtp_port"`
	SMTPUser     string `json:"smtp_user" bson:"smtp_user"`
	SMTPPassword string `json:"smtp_password" bson:"smtp_password"`
	FromEmail    string `json:"from_email" bson:"from_email"`
	FromName     string `json:"from_name" bson:"from_name"`
	Secure       bool   `json:"secure" bson:"secure"` // TLS/SSL
}

type GeneralConfig struct {
	AppName          string `json:"app_name" bson:"app_name"`
	AppURL           string `json:"app_url" bson:"app_url"`
	LogoURL          string `json:"logo_url" bson:"logo_url"`
	Description      string `json:"description" bson:"description"`
	SupportEmail     string `json:"support_email" bson:"support_email"`
	LandingPageTitle string `json:"landing_page_title" bson:"landing_page_title"`
}

type FileSharingConfig struct {
	Enabled              bool     `json:"enabled" bson:"enabled"`
	MaxFileSizeMB        int      `json:"max_file_size_mb" bson:"max_file_size_mb"`
	AllowedFileTypes     []string `json:"allowed_file_types" bson:"allowed_file_types"`
	MaxFilesPerRecord    int      `json:"max_files_per_record" bson:"max_files_per_record"`
	EnabledModules       []string `json:"enabled_modules" bson:"enabled_modules"` // Empty = all modules
	AllowSharedDocuments bool     `json:"allow_shared_documents" bson:"allow_shared_documents"`
}

type Settings struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Type        SettingsType       `json:"type" bson:"type"` // Unique index on type
	Email       *EmailConfig       `json:"email,omitempty" bson:"email,omitempty"`
	General     *GeneralConfig     `json:"general,omitempty" bson:"general,omitempty"`
	FileSharing *FileSharingConfig `json:"file_sharing,omitempty" bson:"file_sharing,omitempty"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}
