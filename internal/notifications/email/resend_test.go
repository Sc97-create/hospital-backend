package email

import (
	"context"
	"errors"
	"strings"
	"testing"

	"hospital-backend/config"
	"hospital-backend/internal/notifications/dto"

	"github.com/resend/resend-go/v2"
)

func TestNewResendProviderRequiresAPIKey(t *testing.T) {
	_, err := NewResendProvider(config.NotificationConfig{FromEmail: "support@example.com"})
	if err == nil || !strings.Contains(err.Error(), "api key") {
		t.Fatalf("err=%v", err)
	}
}

func TestNewResendProviderCreatesClient(t *testing.T) {
	provider, err := NewResendProvider(config.NotificationConfig{
		ResendAPIKey: "re_test_key",
		FromName:     "Floato",
		FromEmail:    "support@example.com",
		ReplyTo:      "reply@example.com",
	})
	if err != nil {
		t.Fatalf("NewResendProvider: %v", err)
	}
	if provider.emails == nil {
		t.Fatal("expected resend client")
	}
	if provider.fromName != "Floato" || provider.fromEmail != "support@example.com" || provider.replyTo != "reply@example.com" {
		t.Fatalf("name=%q from=%q reply=%q", provider.fromName, provider.fromEmail, provider.replyTo)
	}
}

func TestNewResendProviderUsesSMTPUsernameWhenFromEmailEmpty(t *testing.T) {
	provider, err := NewResendProvider(config.NotificationConfig{
		ResendAPIKey: "re_test_key",
		SMTPUsername: "login@example.com",
	})
	if err != nil {
		t.Fatalf("NewResendProvider: %v", err)
	}
	if provider.fromEmail != "login@example.com" {
		t.Fatalf("from=%q", provider.fromEmail)
	}
}

func TestNewResendProviderRequiresFromEmail(t *testing.T) {
	_, err := NewResendProvider(config.NotificationConfig{ResendAPIKey: "re_test_key"})
	if err == nil || !strings.Contains(err.Error(), "from email") {
		t.Fatalf("err=%v", err)
	}
}

func TestResendSendBuildsRequest(t *testing.T) {
	stub := &stubResendEmails{}
	provider, err := newResendProvider(stub, config.NotificationConfig{
		FromName:  "Floato",
		FromEmail: "support@example.com",
		ReplyTo:   "reply@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = provider.Send(context.Background(), dto.Request{
		Recipient: " patient@example.com ",
		Subject:   "Appointment",
		Content:   "<p>See you tomorrow</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := stub.params
	if got == nil {
		t.Fatal("expected send request")
	}
	if got.From != "Floato <support@example.com>" {
		t.Fatalf("from=%q", got.From)
	}
	if len(got.To) != 1 || got.To[0] != "patient@example.com" {
		t.Fatalf("to=%v", got.To)
	}
	if got.Subject != "Appointment" || got.Html != "<p>See you tomorrow</p>" || got.ReplyTo != "reply@example.com" {
		t.Fatalf("subject=%q html=%q reply=%q", got.Subject, got.Html, got.ReplyTo)
	}
}

func TestResendSendOmitsEmptyReplyTo(t *testing.T) {
	stub := &stubResendEmails{}
	provider, err := newResendProvider(stub, config.NotificationConfig{FromEmail: "support@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err = provider.Send(context.Background(), dto.Request{
		Recipient: "patient@example.com",
		Subject:   "Hello",
		Content:   "Hello",
	}); err != nil {
		t.Fatal(err)
	}
	if stub.params.From != "support@example.com" || stub.params.ReplyTo != "" {
		t.Fatalf("from=%q reply=%q", stub.params.From, stub.params.ReplyTo)
	}
}

func TestResendSendRequiresRecipient(t *testing.T) {
	stub := &stubResendEmails{}
	provider, err := newResendProvider(stub, config.NotificationConfig{FromEmail: "support@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err = provider.Send(context.Background(), dto.Request{Subject: "Hello"}); err == nil {
		t.Fatal("expected error")
	}
	if stub.params != nil {
		t.Fatal("client should not be called")
	}
}

func TestResendSendWrapsClientError(t *testing.T) {
	stub := &stubResendEmails{err: errors.New("api down")}
	provider, err := newResendProvider(stub, config.NotificationConfig{FromEmail: "support@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	err = provider.Send(context.Background(), dto.Request{Recipient: "patient@example.com"})
	if err == nil || !strings.Contains(err.Error(), "api down") {
		t.Fatalf("err=%v", err)
	}
}

type stubResendEmails struct {
	params *resend.SendEmailRequest
	err    error
}

func (s *stubResendEmails) SendWithContext(_ context.Context, params *resend.SendEmailRequest) (*resend.SendEmailResponse, error) {
	s.params = params
	if s.err != nil {
		return nil, s.err
	}
	return &resend.SendEmailResponse{Id: "email_123"}, nil
}
