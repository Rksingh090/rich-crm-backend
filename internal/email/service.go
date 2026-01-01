package emails

import (
	"context"
	"errors"
)

type Service struct {
	repo *Repository
	smtp SMTPConfig
}

func NewService(repo *Repository, smtp SMTPConfig) *Service {
	return &Service{repo: repo, smtp: smtp}
}

type SendEmailInput struct {
	OrgID      string
	From       string
	To         []string
	Subject    string
	HtmlBody   string
	EntityType string
	EntityID   string
}

func (s *Service) Send(ctx context.Context, email *Email) error {
	if len(email.To) == 0 {
		return errors.New("recipient required")
	}

	email.Status = EmailQueued
	if err := s.repo.Create(ctx, email); err != nil {
		return err
	}

	go s.process(email)
	return nil
}

func (s *Service) process(email *Email) {
	err := SendSMTP(s.smtp, email)
	if err != nil {
		_ = s.repo.UpdateStatus(
			context.Background(),
			email.ID,
			EmailFailed,
			err.Error(),
		)
		return
	}

	_ = s.repo.UpdateStatus(
		context.Background(),
		email.ID,
		EmailSent,
		"",
	)
}
