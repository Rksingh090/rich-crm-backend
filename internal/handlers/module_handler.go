package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/go-chi/chi/v5"
)

type ModuleHandler struct {
	Service service.ModuleService
}

func NewModuleHandler(service service.ModuleService) *ModuleHandler {
	return &ModuleHandler{
		Service: service,
	}
}

// CreateModule godoc
// @Summary      Create a new module definition
// @Description  Create a dynamic module with fields
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        input body models.Module true "Module Definition"
// @Success      201  {string} string "Module created"
// @Failure      400  {string} string "Invalid request"
// @Router       /modules [post]
func (h *ModuleHandler) CreateModule(w http.ResponseWriter, r *http.Request) {
	var module models.Module
	if err := json.NewDecoder(r.Body).Decode(&module); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.Service.CreateModule(r.Context(), &module); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Module created successfully"})
}

// ListModules godoc
// @Summary      List all modules
// @Description  Get a list of all defined modules
// @Tags         modules
// @Produce      json
// @Success      200  {array} models.Module
// @Router       /modules [get]
func (h *ModuleHandler) ListModules(w http.ResponseWriter, r *http.Request) {
	modules, err := h.Service.ListModules(r.Context())
	if err != nil {
		http.Error(w, "Failed to fetch modules", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(modules)
}

// GetModule godoc
// @Summary      Get a module by name
// @Description  Get specific module definition including fields
// @Tags         modules
// @Produce      json
// @Param        name path string true "Module Name"
// @Success      200  {object} models.Module
// @Failure      404  {string} string "Module not found"
// @Router       /modules/{name} [get]
func (h *ModuleHandler) GetModule(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	module, err := h.Service.GetModuleByName(r.Context(), name)
	if err != nil {
		http.Error(w, "Module not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(module)
}

// UpdateModule godoc
// @Summary      Update a module definition
// @Description  Update schema for a module
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        name path string true "Module Name"
// @Param        input body models.Module true "Module Definition"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid input"
// @Router       /modules/{name} [put]
func (h *ModuleHandler) UpdateModule(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var module models.Module
	if err := json.NewDecoder(r.Body).Decode(&module); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	module.Name = name // Ensure name matches path

	if err := h.Service.UpdateModule(r.Context(), &module); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Module updated successfully"})
}

// DeleteModule godoc
// @Summary      Delete a module
// @Description  Delete a module and its definition
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        name path string true "Module Name"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid input"
// @Router       /modules/{name} [delete]
func (h *ModuleHandler) DeleteModule(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	name := pathParts[2]

	if err := h.Service.DeleteModule(r.Context(), name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Module deleted successfully"})
}
