package file

import (
	"context"
	"fmt"
	"os"

	"go-crm/internal/features/settings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FileService interface {
	GetFilesByRecord(ctx context.Context, moduleName, recordID string) ([]*File, error)
	GetSharedFiles(ctx context.Context) ([]*File, error)
	GetFile(ctx context.Context, fileID string) (*File, error)
	DeleteFile(ctx context.Context, fileID string, userID primitive.ObjectID) error
	ValidateUpload(ctx context.Context, moduleName string, recordID string, fileSize int64, mimeType string) error
	SaveFile(ctx context.Context, file *File) error
}

type FileServiceImpl struct {
	FileRepo     FileRepository
	SettingsRepo settings.SettingsRepository
}

func NewFileService(fileRepo FileRepository, settingsRepo settings.SettingsRepository) FileService {
	return &FileServiceImpl{
		FileRepo:     fileRepo,
		SettingsRepo: settingsRepo,
	}
}

func (s *FileServiceImpl) GetFilesByRecord(ctx context.Context, moduleName, recordID string) ([]*File, error) {
	return s.FileRepo.FindByRecord(ctx, moduleName, recordID)
}

func (s *FileServiceImpl) GetSharedFiles(ctx context.Context) ([]*File, error) {
	return s.FileRepo.FindShared(ctx)
}

func (s *FileServiceImpl) GetFile(ctx context.Context, fileID string) (*File, error) {
	return s.FileRepo.Get(ctx, fileID)
}

func (s *FileServiceImpl) SaveFile(ctx context.Context, file *File) error {
	return s.FileRepo.Save(ctx, file)
}

func (s *FileServiceImpl) DeleteFile(ctx context.Context, fileID string, userID primitive.ObjectID) error {
	file, err := s.FileRepo.Get(ctx, fileID)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	if file.UploadedBy != userID {
		return fmt.Errorf("unauthorized: you can only delete your own files")
	}

	if err := os.Remove(file.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file from disk: %w", err)
	}

	return s.FileRepo.Delete(ctx, fileID)
}

func (s *FileServiceImpl) ValidateUpload(ctx context.Context, moduleName string, recordID string, fileSize int64, mimeType string) error {
	settingsObj, err := s.SettingsRepo.GetByType(ctx, settings.SettingsTypeFileSharing)
	if err != nil {
		return s.validateWithDefaults(moduleName, recordID, fileSize, mimeType)
	}

	if settingsObj == nil || settingsObj.FileSharing == nil {
		return s.validateWithDefaults(moduleName, recordID, fileSize, mimeType)
	}

	config := settingsObj.FileSharing

	if !config.Enabled {
		return fmt.Errorf("file sharing is disabled")
	}

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

	maxSizeBytes := int64(config.MaxFileSizeMB) << 20
	if fileSize > maxSizeBytes {
		return fmt.Errorf("file too large (max %dMB)", config.MaxFileSizeMB)
	}

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
	if fileSize > 10<<20 {
		return fmt.Errorf("file too large (max 10MB)")
	}
	return nil
}

func checkFileExtension(mimeType, allowedType string) bool {
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
