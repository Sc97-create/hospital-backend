package email

import (
	"context"
	"fmt"
	"hospital-backend/config"
	"hospital-backend/internal/notifications/dto"

	"github.com/wneessen/go-mail"
)

type SMTPProvider struct {
	client    *mail.Client
	fromEmail string
	fromName  string
}

func NewSmtpProvider(cfg config.NotificationConfig) (*SMTPProvider, error) {
	// Create a new client with SMTP server settings
	client, err := mail.NewClient(
		cfg.SMTPHost,
		mail.WithPort(cfg.SMTPPort),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(cfg.SMTPUsername),
		mail.WithPassword(cfg.SMTPPassword),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	return &SMTPProvider{
		client:    client,
		fromEmail: cfg.SMTPUsername,
		fromName:  cfg.FromName,
	}, nil
}

func (s *SMTPProvider) Send(ctx context.Context, req dto.Request) error {
	msg := mail.NewMsg()

	if err := msg.FromFormat(s.fromName, s.fromEmail); err != nil {
		return fmt.Errorf("failed to set from address: %w", err)
	}

	if err := msg.To(req.Recipient); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	msg.Subject(req.Subject)

	msg.SetBodyString(mail.TypeTextHTML, req.Content)

	return s.client.DialAndSend(msg)
}
