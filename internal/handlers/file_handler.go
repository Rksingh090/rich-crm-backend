package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-crm/internal/models"
	"go-crm/internal/repository"
)

type FileHandler struct {
	UploadDir string
	Repo      repository.FileRepository
}

func NewFileHandler(uploadDir string, repo repository.FileRepository) *FileHandler {
	// Ensure upload directory exists
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}
	return &FileHandler{
		UploadDir: uploadDir,
		Repo:      repo,
	}
}

// UploadFile godoc
// @Summary      Upload a file
// @Description  Upload a file and get a URL (Metadata stored in DB)
// @Tags         files
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData file   true  "File to upload"
// @Param        group formData string false "Group (optional)"
// @Success      200   {object} models.File
// @Failure      400   {string} string "Invalid input"
// @Failure      500   {string} string "Internal Server Error"
// @Router       /upload [post]
func (h *FileHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	// 1. Parse Multipart Form (10 MB limit)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "File too large (max 10MB)", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 2. Validate Extension (Optional)
	// ext := filepath.Ext(handler.Filename)

	// 3. Create Unique Filename
	originalName := filepath.Base(handler.Filename)
	timestamp := time.Now().UnixNano()
	uniqueName := fmt.Sprintf("%d_%s", timestamp, originalName)
	uniqueName = strings.ReplaceAll(uniqueName, " ", "_")

	dstPath := filepath.Join(h.UploadDir, uniqueName)

	// 4. Save File to Disk
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Error saving file to disk", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Error saving file content", http.StatusInternalServerError)
		return
	}

	// 5. Save Metadata to DB
	fileRecord := &models.File{
		OriginalFilename: originalName,
		UniqueFilename:   uniqueName,
		Path:             dstPath,
		URL:              fmt.Sprintf("/uploads/%s", uniqueName),
		Group:            r.FormValue("group"),
		Size:             size,
		MIMEType:         handler.Header.Get("Content-Type"),
		CreatedAt:        time.Now(),
	}

	if err := h.Repo.Save(r.Context(), fileRecord); err != nil {
		http.Error(w, "Error saving file metadata", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(fileRecord)
}
