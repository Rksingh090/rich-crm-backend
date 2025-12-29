package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go-crm/internal/models"
	"go-crm/internal/repository"
)

type WebhookService interface {
	CreateWebhook(ctx context.Context, webhook *models.Webhook) error
	ListWebhooks(ctx context.Context) ([]models.Webhook, error)
	GetWebhook(ctx context.Context, id string) (*models.Webhook, error)
	UpdateWebhook(ctx context.Context, id string, updates map[string]interface{}) error
	DeleteWebhook(ctx context.Context, id string) error
	Trigger(ctx context.Context, event string, payload models.WebhookPayload)
}

type WebhookServiceImpl struct {
	Repo         repository.WebhookRepository
	AuditService AuditService
	HttpClient   *http.Client
}

func NewWebhookService(repo repository.WebhookRepository, auditService AuditService) WebhookService {
	return &WebhookServiceImpl{
		Repo:         repo,
		AuditService: auditService,
		HttpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *WebhookServiceImpl) CreateWebhook(ctx context.Context, webhook *models.Webhook) error {
	err := s.Repo.Create(ctx, webhook)
	if err == nil {
		s.AuditService.LogChange(ctx, models.AuditActionWebhook, "webhooks", webhook.ID.Hex(), map[string]models.Change{
			"webhook": {New: webhook},
		})
	}
	return err
}

func (s *WebhookServiceImpl) ListWebhooks(ctx context.Context) ([]models.Webhook, error) {
	return s.Repo.List(ctx)
}

func (s *WebhookServiceImpl) GetWebhook(ctx context.Context, id string) (*models.Webhook, error) {
	return s.Repo.Get(ctx, id)
}

func (s *WebhookServiceImpl) UpdateWebhook(ctx context.Context, id string, updates map[string]interface{}) error {
	// Get old webhook for audit
	oldWebhook, _ := s.GetWebhook(ctx, id)

	err := s.Repo.Update(ctx, id, updates)
	if err == nil {
		s.AuditService.LogChange(ctx, models.AuditActionWebhook, "webhooks", id, map[string]models.Change{
			"webhook": {Old: oldWebhook, New: updates},
		})
	}
	return err
}

func (s *WebhookServiceImpl) DeleteWebhook(ctx context.Context, id string) error {
	// Get old webhook for audit
	oldWebhook, _ := s.GetWebhook(ctx, id)

	err := s.Repo.Delete(ctx, id)
	if err == nil {
		name := id
		if oldWebhook != nil {
			name = oldWebhook.URL
		}
		s.AuditService.LogChange(ctx, models.AuditActionWebhook, "webhooks", name, map[string]models.Change{
			"webhook": {Old: oldWebhook, New: "DELETED"},
		})
	}
	return err
}

func (s *WebhookServiceImpl) Trigger(ctx context.Context, event string, payload models.WebhookPayload) {
	// Find subscribers
	// Note: We use a detached context or background for the DB lookup?
	// The Trigger call is likely inside a request scope, so DB lookup works.
	// But the HTTP sending should be async.

	webhooks, err := s.Repo.ListByEvent(ctx, event)
	if err != nil {
		fmt.Printf("Error fetching webhooks for event %s: %v\n", event, err)
		return
	}

	for _, wh := range webhooks {
		// Filter by module if specified
		if wh.ModuleName != "" && wh.ModuleName != payload.Module {
			continue
		}

		go s.sendWebhook(wh, payload)
	}
}

func (s *WebhookServiceImpl) sendWebhook(wh models.Webhook, payload models.WebhookPayload) {
	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshalling webhook payload: %v\n", err)
		return
	}

	req, err := http.NewRequest("POST", wh.URL, bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("Error creating webhook request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Go-CRM-Webhook")
	req.Header.Set("X-CRM-Event", payload.Event)
	req.Header.Set("X-CRM-Delivery", fmt.Sprintf("%d", time.Now().UnixNano()))

	// Custom Headers
	for k, v := range wh.Headers {
		req.Header.Set(k, v)
	}

	// Signature
	if wh.Secret != "" {
		mac := hmac.New(sha256.New, []byte(wh.Secret))
		mac.Write(body)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-CRM-Signature", "sha256="+signature)
	}

	resp, err := s.HttpClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending webhook to %s: %v\n", wh.URL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("Webhook to %s failed with status: %d\n", wh.URL, resp.StatusCode)
	}
}
