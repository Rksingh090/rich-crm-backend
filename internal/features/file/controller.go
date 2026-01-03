package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-crm/internal/config"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FileController struct {
	UploadDir   string
	FileService FileService
	Config      *config.Config
}

func NewFileController(fileService FileService, cfg *config.Config) *FileController {
	if _, err := os.Stat(cfg.FSPath); os.IsNotExist(err) {
		os.MkdirAll(cfg.FSPath, 0755)
	}
	return &FileController{
		UploadDir:   cfg.FSPath,
		FileService: fileService,
		Config:      cfg,
	}
}

func (ctrl *FileController) UploadFile(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error retrieving file",
		})
	}

	moduleName := c.FormValue("module_name")
	recordID := c.FormValue("record_id")
	isShared := c.FormValue("is_shared") == "true"
	description := c.FormValue("description")

	if err := ctrl.FileService.ValidateUpload(c.UserContext(), moduleName, recordID, file.Size, file.Header.Get("Content-Type")); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	originalName := filepath.Base(file.Filename)
	timestamp := time.Now().UnixNano()
	uniqueName := fmt.Sprintf("%d_%s", timestamp, originalName)
	uniqueName = strings.ReplaceAll(uniqueName, " ", "_")

	dstPath := filepath.Join(ctrl.UploadDir, uniqueName)

	if err := c.SaveFile(file, dstPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error saving file to disk",
		})
	}

	fileRecord := &File{
		OriginalFilename: originalName,
		Path:             dstPath,
		URL:              "/api/files/download/" + uniqueName, // Basic URL, should ideally be configurable
		Size:             file.Size,
		MimeType:         file.Header.Get("Content-Type"),
		ModuleName:       moduleName,
		RecordID:         recordID,
		UploadedBy:       userID,
		IsShared:         isShared,
		StorageType:      "local",
		Description:      description,
		CreatedAt:        time.Now(),
	}

	if err := ctrl.FileService.SaveFile(c.UserContext(), fileRecord); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error saving file metadata",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fileRecord)
}

func (ctrl *FileController) GetFilesByRecord(c *fiber.Ctx) error {
	moduleName := c.Params("module")
	recordID := c.Params("recordId")

	files, err := ctrl.FileService.GetFilesByRecord(c.UserContext(), moduleName, recordID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error retrieving files",
		})
	}

	return c.JSON(files)
}

func (ctrl *FileController) GetSharedFiles(c *fiber.Ctx) error {
	files, err := ctrl.FileService.GetSharedFiles(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error retrieving shared files",
		})
	}

	return c.JSON(files)
}

func (ctrl *FileController) DownloadFile(c *fiber.Ctx) error {
	fileID := c.Params("id")

	file, err := ctrl.FileService.GetFile(c.UserContext(), fileID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	return c.Download(file.Path, file.OriginalFilename)
}

func (ctrl *FileController) DeleteFile(c *fiber.Ctx) error {
	fileID := c.Params("id")

	userIDStr := c.Locals("user_id").(string)
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	if err := ctrl.FileService.DeleteFile(c.UserContext(), fileID, userID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "File deleted successfully",
	})
}
