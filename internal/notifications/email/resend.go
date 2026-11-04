package email

import (
	"context"
	"fmt"
	"strings"

	"hospital-backend/config"
	"hospital-backend/internal/notifications/dto"

	"github.com/resend/resend-go/v2"
)

type resendEmails interface {
	SendWithContext(ctx context.Context, params *resend.SendEmailRequest) (*resend.SendEmailResponse, error)
}

type ResendProvider struct {
	emails    resendEmails
	fromEmail string
	fromName  string
	replyTo   string
}

func NewResendProvider(cfg config.NotificationConfig) (*ResendProvider, error) {
	apiKey := strings.TrimSpace(cfg.ResendAPIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("resend api key is required")
	}
	client := resend.NewClient(apiKey)
	return newResendProvider(client.Emails, cfg)
}

func newResendProvider(emails resendEmails, cfg config.NotificationConfig) (*ResendProvider, error) {
	fromName, fromEmail, replyTo := mailIdentity(cfg)
	if fromEmail == "" {
		return nil, fmt.Errorf("failed to set from address: from email is required")
	}
	return &ResendProvider{
		emails:    emails,
		fromEmail: fromEmail,
		fromName:  fromName,
		replyTo:   replyTo,
	}, nil
}

func (r *ResendProvider) Send(ctx context.Context, req dto.Request) error {
	if strings.TrimSpace(req.Recipient) == "" {
		return fmt.Errorf("failed to set recipient: recipient is required")
	}
	if _, err := r.emails.SendWithContext(ctx, r.sendParams(req)); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}

func (r *ResendProvider) sendParams(req dto.Request) *resend.SendEmailRequest {
	params := &resend.SendEmailRequest{
		From:    resendFromAddress(r.fromName, r.fromEmail),
		To:      []string{strings.TrimSpace(req.Recipient)},
		Subject: req.Subject,
		Html:    req.Content,
	}
	if r.replyTo != "" {
		params.ReplyTo = r.replyTo
	}
	return params
}

func resendFromAddress(name, email string) string {
	if name == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", name, email)
}
