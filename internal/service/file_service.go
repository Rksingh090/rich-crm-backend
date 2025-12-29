package service

import (
	"context"
	"fmt"
	"os"

	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FileService interface {
	GetFilesByRecord(ctx context.Context, moduleName, recordID string) ([]*models.File, error)
	GetSharedFiles(ctx context.Context) ([]*models.File, error)
	GetFile(ctx context.Context, fileID string) (*models.File, error)
	DeleteFile(ctx context.Context, fileID string, userID primitive.ObjectID) error
	ValidateUpload(ctx context.Context, moduleName string, recordID string, fileSize int64, mimeType string) error
}

type FileServiceImpl struct {
	FileRepo     repository.FileRepository
	SettingsRepo repository.SettingsRepository
}

func NewFileService(fileRepo repository.FileRepository, settingsRepo repository.SettingsRepository) FileService {
	return &FileServiceImpl{
		FileRepo:     fileRepo,
		SettingsRepo: settingsRepo,
	}
}

func (s *FileServiceImpl) GetFilesByRecord(ctx context.Context, moduleName, recordID string) ([]*models.File, error) {
	return s.FileRepo.FindByRecord(ctx, moduleName, recordID)
}

func (s *FileServiceImpl) GetSharedFiles(ctx context.Context) ([]*models.File, error) {
	return s.FileRepo.FindShared(ctx)
}

func (s *FileServiceImpl) GetFile(ctx context.Context, fileID string) (*models.File, error) {
	return s.FileRepo.Get(ctx, fileID)
}

func (s *FileServiceImpl) DeleteFile(ctx context.Context, fileID string, userID primitive.ObjectID) error {
	// Get file to check ownership and delete from disk
	file, err := s.FileRepo.Get(ctx, fileID)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// Check if user owns the file
	if file.UploadedBy != userID {
		return fmt.Errorf("unauthorized: you can only delete your own files")
	}

	// Delete file from disk
	if err := os.Remove(file.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file from disk: %w", err)
	}

	// Delete from database
	return s.FileRepo.Delete(ctx, fileID)
}

func (s *FileServiceImpl) ValidateUpload(ctx context.Context, moduleName string, recordID string, fileSize int64, mimeType string) error {
	// Get file sharing settings
	settings, err := s.SettingsRepo.GetByType(ctx, models.SettingsTypeFileSharing)
	if err != nil {
		// If settings not configured, use defaults
		return s.validateWithDefaults(moduleName, recordID, fileSize, mimeType)
	}

	if settings == nil || settings.FileSharing == nil {
		return s.validateWithDefaults(moduleName, recordID, fileSize, mimeType)
	}

	config := settings.FileSharing

	// Check if file sharing is enabled
	if !config.Enabled {
		return fmt.Errorf("file sharing is disabled")
	}

	// Check if module is enabled (empty list = all modules enabled)
	if len(config.EnabledModules) > 0 && moduleName != "" {
		moduleEnabled := false
		for _, mod := range config.EnabledModules {
			if mod == moduleName {
				moduleEnabled = true
				break
			}
		}
		if !moduleEnabled {
			return fmt.Errorf("file sharing is disabled for module: %s", moduleName)
		}
	}

	// Check file size
	maxSizeBytes := int64(config.MaxFileSizeMB) << 20 // Convert MB to bytes
	if fileSize > maxSizeBytes {
		return fmt.Errorf("file too large (max %dMB)", config.MaxFileSizeMB)
	}

	// Check file type
	if len(config.AllowedFileTypes) > 0 {
		allowed := false
		for _, allowedType := range config.AllowedFileTypes {
			if mimeType == allowedType || checkFileExtension(mimeType, allowedType) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file type not allowed: %s", mimeType)
		}
	}

	// Check max files per record if attaching to a record
	if moduleName != "" && recordID != "" {
		count, err := s.FileRepo.CountByRecord(ctx, moduleName, recordID)
		if err != nil {
			return fmt.Errorf("failed to check file count: %w", err)
		}
		if int(count) >= config.MaxFilesPerRecord {
			return fmt.Errorf("maximum files per record reached (%d)", config.MaxFilesPerRecord)
		}
	}

	return nil
}

func (s *FileServiceImpl) validateWithDefaults(moduleName string, recordID string, fileSize int64, mimeType string) error {
	// Default: 10MB limit
	if fileSize > 10<<20 {
		return fmt.Errorf("file too large (max 10MB)")
	}
	return nil
}

func checkFileExtension(mimeType, allowedType string) bool {
	// Map common MIME types to extensions
	mimeToExt := map[string][]string{
		"application/pdf":    {".pdf"},
		"application/msword": {".doc"},
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": {".docx"},
		"application/vnd.ms-excel": {".xls"},
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": {".xlsx"},
		"image/png":  {".png"},
		"image/jpeg": {".jpg", ".jpeg"},
		"image/gif":  {".gif"},
		"text/plain": {".txt"},
		"text/csv":   {".csv"},
	}

	if exts, ok := mimeToExt[mimeType]; ok {
		for _, ext := range exts {
			if ext == allowedType {
				return true
			}
		}
	}
	return false
}
