package handlers

import (
	"encoding/json"
	"go-crm/internal/service"
	"net/http"
	"strconv"
)

type AuditHandler struct {
	Service service.AuditService
}

func NewAuditHandler(service service.AuditService) *AuditHandler {
	return &AuditHandler{Service: service}
}

// ListLogs godoc
// @Summary      List audit logs
// @Description  Get audit logs with pagination
// @Tags         audit
// @Accept       json
// @Produce      json
// @Param        page  query int    false "Page number (default 1)"
// @Param        limit query int    false "Logs per page (default 20)"
// @Success      200   {array}  models.AuditLog
// @Failure      500   {string} string "Internal Server Error"
// @Router       /audit-logs [get]
func (h *AuditHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, _ := strconv.ParseInt(pageStr, 10, 64)
	limit, _ := strconv.ParseInt(limitStr, 10, 64)

	if limit == 0 {
		limit = 20
	}

	logs, err := h.Service.ListLogs(r.Context(), page, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(logs)
}
