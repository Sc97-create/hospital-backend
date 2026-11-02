package email

import (
	"context"
	"fmt"
	"hospital-backend/config"
	"hospital-backend/internal/notifications/dto"
	"strings"
	"time"

	"github.com/wneessen/go-mail"
)

type SMTPProvider struct {
	client    *mail.Client
	fromEmail string
	fromName  string
	replyTo   string
}

func NewSmtpProvider(cfg config.NotificationConfig) (*SMTPProvider, error) {
	fromName, fromEmail, replyTo := mailIdentity(cfg)
	client, err := mail.NewClient(cfg.SMTPHost, smtpClientOptions(cfg)...)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	return &SMTPProvider{
		client:    client,
		fromEmail: fromEmail,
		fromName:  fromName,
		replyTo:   replyTo,
	}, nil
}

func smtpClientOptions(cfg config.NotificationConfig) []mail.Option {
	opts := []mail.Option{
		mail.WithPort(cfg.SMTPPort),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(cfg.SMTPUsername),
		mail.WithPassword(cfg.SMTPPassword),
		mail.WithTimeout(60 * time.Second),
	}
	if cfg.SMTPPort == 465 {
		opts = append(opts, mail.WithSSL())
	}
	return opts
}

func mailIdentity(cfg config.NotificationConfig) (name, email, replyTo string) {
	email = strings.TrimSpace(cfg.FromEmail)
	if email == "" {
		email = strings.TrimSpace(cfg.SMTPUsername)
	}
	return strings.TrimSpace(cfg.FromName), email, strings.TrimSpace(cfg.ReplyTo)
}

func (s *SMTPProvider) Send(ctx context.Context, req dto.Request) error {
	msg := mail.NewMsg()

	if err := s.setFrom(msg); err != nil {
		return err
	}
	if err := s.setReplyTo(msg); err != nil {
		return err
	}
	if err := msg.To(req.Recipient); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	msg.Subject(req.Subject)
	msg.SetBodyString(mail.TypeTextHTML, req.Content)

	return s.client.DialAndSend(msg)
}

func (s *SMTPProvider) setFrom(msg *mail.Msg) error {
	if s.fromName == "" {
		if err := msg.From(s.fromEmail); err != nil {
			return fmt.Errorf("failed to set from address: %w", err)
		}
		return nil
	}
	if err := msg.FromFormat(s.fromName, s.fromEmail); err != nil {
		return fmt.Errorf("failed to set from address: %w", err)
	}
	return nil
}

func (s *SMTPProvider) setReplyTo(msg *mail.Msg) error {
	if s.replyTo == "" {
		return nil
	}
	if err := msg.ReplyTo(s.replyTo); err != nil {
		return fmt.Errorf("failed to set reply-to: %w", err)
	}
	return nil
}
