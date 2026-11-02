package email

import (
	"testing"

	"hospital-backend/config"
)

func TestMailIdentityUsesMailFields(t *testing.T) {
	name, from, reply := mailIdentity(config.NotificationConfig{
		SMTPUsername: "login@example.com",
		FromName:     "Floato",
		FromEmail:    "support@example.com",
		ReplyTo:      "support@example.com",
	})
	if name != "Floato" || from != "support@example.com" || reply != "support@example.com" {
		t.Fatalf("name=%q from=%q reply=%q", name, from, reply)
	}
}

func TestMailIdentityFallsBackToSMTPUsername(t *testing.T) {
	_, from, _ := mailIdentity(config.NotificationConfig{SMTPUsername: "login@example.com"})
	if from != "login@example.com" {
		t.Fatalf("from=%q", from)
	}
}

func TestSMTPPort465EnablesSSL(t *testing.T) {
	opts := smtpClientOptions(config.NotificationConfig{
		SMTPPort:     465,
		SMTPUsername: "login@example.com",
		SMTPPassword: "secret",
	})
	if len(opts) != 6 {
		t.Fatalf("expected SSL option on port 465, got %d options", len(opts))
	}
}
