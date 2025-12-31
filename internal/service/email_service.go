package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/smtp"
	"path/filepath"
)

type EmailService interface {
	SendEmail(ctx context.Context, to []string, subject, body string) error
	SendEmailWithAttachment(ctx context.Context, to []string, subject, body string, attachmentName string, attachmentData []byte) error
}

type EmailServiceImpl struct {
	SettingsService SettingsService
}

func NewEmailService(settingsService SettingsService) EmailService {
	return &EmailServiceImpl{
		SettingsService: settingsService,
	}
}

func (s *EmailServiceImpl) SendEmail(ctx context.Context, to []string, subject, body string) error {
	// 1. Fetch Config
	config, err := s.SettingsService.GetEmailConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch email config: %v", err)
	}
	if config == nil {
		return errors.New("email configuration not found")
	}

	// 2. Validate Config
	if config.SMTPHost == "" || config.SMTPPort == 0 {
		return errors.New("invalid email configuration: missing host or port")
	}

	// 3. Prepare Auth
	auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPassword, config.SMTPHost)

	// 4. Prepare Message
	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
	from := config.FromEmail
	if from == "" {
		from = config.SMTPUser // Fallback
	}

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", to[0], subject, body))

	// 5. Send
	log.Printf("Sending email to %v via %s...", to, addr)
	err = smtp.SendMail(addr, auth, from, to, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Println("Email sent successfully")
	return nil
}

func (s *EmailServiceImpl) SendEmailWithAttachment(ctx context.Context, to []string, subject, body string, attachmentName string, attachmentData []byte) error {
	// 1. Fetch Config
	config, err := s.SettingsService.GetEmailConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch email config: %v", err)
	}
	if config == nil {
		return errors.New("email configuration not found")
	}

	// 2. Validate Config
	if config.SMTPHost == "" || config.SMTPPort == 0 {
		return errors.New("invalid email configuration: missing host or port")
	}

	// 3. Prepare Auth
	auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPassword, config.SMTPHost)

	// 4. Prepare Message with MIME multipart for attachment
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

	// Body
	buf.WriteString(fmt.Sprintf("--%s\r\n", marker))
	buf.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(body)
	buf.WriteString("\r\n")

	// Attachment
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

	// 5. Send
	log.Printf("Sending email with attachment to %v via %s...", to, addr)
	err = smtp.SendMail(addr, auth, from, to, buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send email with attachment: %v", err)
	}

	log.Println("Email with attachment sent successfully")
	return nil
}
