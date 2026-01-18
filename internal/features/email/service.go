package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/settings"
	"log"
	"mime"
	"net/smtp"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EmailService interface {
	SendEmail(ctx context.Context, to []string, subject, body string) error
	SendEmailWithAttachment(ctx context.Context, to []string, subject, body string, attachmentName string, attachmentData []byte) error
	SendEmailWithOptions(ctx context.Context, options EmailOptions) error
}

type EmailServiceImpl struct {
	SettingsService settings.SettingsService
	Repo            *EmailRepository
}

func NewEmailService(settingsService settings.SettingsService, repo *EmailRepository) EmailService {
	return &EmailServiceImpl{
		SettingsService: settingsService,
		Repo:            repo,
	}
}

func (s *EmailServiceImpl) SendEmail(ctx context.Context, to []string, subject, body string) error {
	config, err := s.SettingsService.GetEmailConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch email config: %v", err)
	}
	if config == nil {
		return errors.New("email configuration not found")
	}

	if config.SMTPHost == "" || config.SMTPPort == 0 {
		return errors.New("invalid email configuration: missing host or port")
	}

	auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPassword, config.SMTPHost)

	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
	from := config.FromEmail
	if from == "" {
		from = config.SMTPUser
	}

	log.Printf("DEBUG SMTP: User='%s', From='%s', Host='%s'", config.SMTPUser, from, config.SMTPHost)

	// Try to get TenantID and AppID from context
	var tenantID primitive.ObjectID
	var appID string
	if val := ctx.Value(common_models.TenantIDKey); val != nil {
		if id, ok := val.(string); ok {
			if oid, err := primitive.ObjectIDFromHex(id); err == nil {
				tenantID = oid
			}
		} else if oid, ok := val.(primitive.ObjectID); ok {
			tenantID = oid
		}
	}
	if val := ctx.Value(common_models.AppIDKey); val != nil {
		if id, ok := val.(string); ok {
			appID = id
		}
	}

	// Create email record
	emailRecord := &Email{
		ID:       primitive.NewObjectID(),
		TenantID: tenantID,
		App:      appID,
		From:     from,
		To:       to,
		Subject:  subject,
		HtmlBody: body,
		Status:   EmailQueued,
	}

	if s.Repo != nil {
		_ = s.Repo.Create(ctx, emailRecord)
	}

	// Build email with proper MIME headers for HTML
	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to[0]))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	log.Printf("Sending email to %v via %s...", to, addr)
	err = smtp.SendMail(addr, auth, from, to, msg.Bytes())

	status := EmailSent
	errMsg := ""
	if err != nil {
		status = EmailFailed
		errMsg = err.Error()
	}

	if s.Repo != nil {
		_ = s.Repo.UpdateStatus(ctx, emailRecord.ID, status, errMsg)
	}

	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Println("Email sent successfully")
	return nil
}

func (s *EmailServiceImpl) SendEmailWithOptions(ctx context.Context, opts EmailOptions) error {
	config, err := s.SettingsService.GetEmailConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch email config: %v", err)
	}
	if config == nil {
		return errors.New("email configuration not found")
	}

	if config.SMTPHost == "" || config.SMTPPort == 0 {
		return errors.New("invalid email configuration: missing host or port")
	}

	auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPassword, config.SMTPHost)
	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)

	from := opts.From
	if from == "" {
		from = config.FromEmail
		if from == "" {
			from = config.SMTPUser
		}
	}

	// Try to get TenantID and AppID from context
	var tenantID primitive.ObjectID
	var appID string
	if val := ctx.Value(common_models.TenantIDKey); val != nil {
		if id, ok := val.(string); ok {
			if oid, err := primitive.ObjectIDFromHex(id); err == nil {
				tenantID = oid
			}
		} else if oid, ok := val.(primitive.ObjectID); ok {
			tenantID = oid
		}
	}
	if val := ctx.Value(common_models.AppIDKey); val != nil {
		if id, ok := val.(string); ok {
			appID = id
		}
	}

	// Create email record
	emailRecord := &Email{
		ID:       primitive.NewObjectID(),
		TenantID: tenantID,
		App:      appID,
		From:     from,
		To:       opts.To,
		Cc:       opts.Cc,
		Bcc:      opts.Bcc,
		Subject:  opts.Subject,
		HtmlBody: opts.Body,
		Status:   EmailQueued,
	}

	if s.Repo != nil {
		_ = s.Repo.Create(ctx, emailRecord)
	}

	// Build email with proper MIME headers
	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	if len(opts.To) > 0 {
		msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(opts.To, ", ")))
	}
	if len(opts.Cc) > 0 {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(opts.Cc, ", ")))
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", opts.Subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(opts.Body)

	// Combine all recipients for SendMail but don't include BCC in headers
	allRecipients := append([]string{}, opts.To...)
	allRecipients = append(allRecipients, opts.Cc...)
	allRecipients = append(allRecipients, opts.Bcc...)

	log.Printf("Sending email to %v (CC: %v, BCC: %v) via %s...", opts.To, opts.Cc, opts.Bcc, addr)
	err = smtp.SendMail(addr, auth, from, allRecipients, msg.Bytes())

	status := EmailSent
	errMsg := ""
	if err != nil {
		status = EmailFailed
		errMsg = err.Error()
	}

	if s.Repo != nil {
		_ = s.Repo.UpdateStatus(ctx, emailRecord.ID, status, errMsg)
	}

	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Println("Email sent successfully")
	return nil
}

func (s *EmailServiceImpl) SendEmailWithAttachment(ctx context.Context, to []string, subject, body string, attachmentName string, attachmentData []byte) error {
	config, err := s.SettingsService.GetEmailConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch email config: %v", err)
	}
	if config == nil {
		return errors.New("email configuration not found")
	}

	if config.SMTPHost == "" || config.SMTPPort == 0 {
		return errors.New("invalid email configuration: missing host or port")
	}

	auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPassword, config.SMTPHost)

	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
	from := config.FromEmail
	if from == "" {
		from = config.SMTPUser
	}

	marker := "ACRMarker"
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to[0]))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", marker))
	buf.WriteString("\r\n")

	buf.WriteString(fmt.Sprintf("--%s\r\n", marker))
	buf.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(body)
	buf.WriteString("\r\n")

	if len(attachmentData) > 0 {
		buf.WriteString(fmt.Sprintf("--%s\r\n", marker))
		ext := filepath.Ext(attachmentName)
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		buf.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", contentType, attachmentName))
		buf.WriteString("Content-Transfer-Encoding: base64\r\n")
		buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", attachmentName))
		buf.WriteString("\r\n")

		b := make([]byte, base64.StdEncoding.EncodedLen(len(attachmentData)))
		base64.StdEncoding.Encode(b, attachmentData)
		buf.Write(b)
		buf.WriteString("\r\n")
	}

	buf.WriteString(fmt.Sprintf("--%s--", marker))

	log.Printf("Sending email with attachment to %v via %s...", to, addr)
	err = smtp.SendMail(addr, auth, from, to, buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send email with attachment: %v", err)
	}

	log.Println("Email with attachment sent successfully")
	return nil
}
