package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-crm/internal/service"

	"github.com/go-chi/chi/v5"
)

type RecordHandler struct {
	Service service.RecordService
}

func NewRecordHandler(service service.RecordService) *RecordHandler {
	return &RecordHandler{
		Service: service,
	}
}

// CreateRecord godoc
// @Summary      Create a record in a module
// @Description  Insert data into a dynamic module with validation
// @Tags         records
// @Accept       json
// @Produce      json
// @Param        name path string true "Module Name"
// @Param        input body map[string]interface{} true "Record Data"
// @Success      201  {object} map[string]interface{}
// @Failure      400  {string} string "Invalid request"
// @Failure      404  {string} string "Module not found"
// @Router       /modules/{name}/records [post]
func (h *RecordHandler) CreateRecord(w http.ResponseWriter, r *http.Request) {
	moduleName := chi.URLParam(r, "name")

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	id, err := h.Service.CreateRecord(r.Context(), moduleName, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest) // 400 for validation errors
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Record created successfully",
		"id":      id,
	})
}

// ListRecords godoc
// @Summary      List records from a module
// @Description  Get records with pagination and filters
// @Tags         records
// @Accept       json
// @Produce      json
// @Param        name  path string true "Module Name"
// @Param        page  query int    false "Page number (default 1)"
// @Param        limit query int    false "Records per page (default 10)"
// @Success      200   {array}  map[string]interface{}
// @Failure      400   {string} string "Invalid input"
// @Router       /modules/{name}/records [get]
func (h *RecordHandler) ListRecords(w http.ResponseWriter, r *http.Request) {
	moduleName := chi.URLParam(r, "name")

	query := r.URL.Query()
	pageStr := query.Get("page")
	limitStr := query.Get("limit")

	page, _ := strconv.ParseInt(pageStr, 10, 64)
	limit, _ := strconv.ParseInt(limitStr, 10, 64)

	// Extract everything else as filters
	filters := make(map[string]interface{})
	for k, v := range query {
		if k == "page" || k == "limit" {
			continue
		}
		// For basic filtering, just take the first value
		// Advanced: handle operators (gt, lt) here if keys are suffixed
		if len(v) > 0 {
			filters[k] = v[0]
		}
	}

	records, err := h.Service.ListRecords(r.Context(), moduleName, filters, page, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(records)
}

// GetRecord godoc
// @Summary      Get a single record
// @Description  Get a record by ID
// @Tags         records
// @Produce      json
// @Param        module path string true "Module Name"
// @Param        id     path string true "Record ID"
// @Success      200    {object} map[string]any
// @Failure      404    {string} string "Not Found"
// @Router       /modules/{module}/records/{id} [get]
func (h *RecordHandler) GetRecord(w http.ResponseWriter, r *http.Request) {
	moduleName := chi.URLParam(r, "name")
	id := chi.URLParam(r, "id")

	record, err := h.Service.GetRecord(r.Context(), moduleName, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(record)
}

// UpdateRecord godoc
// @Summary      Update a record
// @Description  Update a record in a module
// @Tags         records
// @Accept       json
// @Produce      json
// @Param        name path string true "Module Name"
// @Param        id   path string true "Record ID"
// @Param        input body map[string]interface{} true "Record Data"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid input"
// @Router       /modules/{name}/records/{id} [put]
func (h *RecordHandler) UpdateRecord(w http.ResponseWriter, r *http.Request) {
	moduleName := chi.URLParam(r, "name")
	id := chi.URLParam(r, "id")

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.Service.UpdateRecord(r.Context(), moduleName, id, data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Record updated successfully"})
}

// DeleteRecord godoc
// @Summary      Delete a record
// @Description  Delete a record from a module
// @Tags         records
// @Accept       json
// @Produce      json
// @Param        name path string true "Module Name"
// @Param        id   path string true "Record ID"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid input"
// @Router       /modules/{name}/records/{id} [delete]
func (h *RecordHandler) DeleteRecord(w http.ResponseWriter, r *http.Request) {
	moduleName := chi.URLParam(r, "name")
	id := chi.URLParam(r, "id")

	if err := h.Service.DeleteRecord(r.Context(), moduleName, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Record deleted successfully"})
}
