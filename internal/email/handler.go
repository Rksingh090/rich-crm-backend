package emails

import (
	"net/http"

	"encoding/json"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type sendEmailRequest struct {
	From       string   `json:"from"`
	To         []string `json:"to"`
	Subject    string   `json:"subject"`
	HtmlBody   string   `json:"htmlBody"`
	EntityType string   `json:"entityType"`
	EntityID   string   `json:"entityId"`
}

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	var req sendEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	orgID := r.Context().Value("orgId").(primitive.ObjectID)

	email := &Email{
		OrgID:      orgID,
		From:       req.From,
		To:         req.To,
		Subject:    req.Subject,
		HtmlBody:   req.HtmlBody,
		EntityType: req.EntityType,
	}

	if err := h.service.Send(r.Context(), email); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(email)
}
