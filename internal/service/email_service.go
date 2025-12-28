package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/smtp"
)

type EmailService interface {
	SendEmail(ctx context.Context, to []string, subject, body string) error
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
	// Note: smtp.SendMail is blocking. For high volume, use a queue.
	// For now, simple synchronous send.
	log.Printf("Sending email to %v via %s...", to, addr)
	err = smtp.SendMail(addr, auth, from, to, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Println("Email sent successfully")
	return nil
}
