package payments

import (
	"testing"

	"hospital-backend/internal/payments/dto"
	"hospital-backend/internal/testutil/servicetest"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type stubPaymentAttemptSvc struct {
	attempt PaymentAttempts
	err     error
}

func (s stubPaymentAttemptSvc) CreateAttempt(_ *zap.Logger, _ *gorm.DB, _ string, _ dto.CreatePaymentResponse, _ string) error {
	return nil
}

func (s stubPaymentAttemptSvc) CreateAttemptWithIdempotency(_ *zap.Logger, _ *gorm.DB, _ string, _ dto.CreatePaymentResponse, _ string, _ string) error {
	return nil
}

func (s stubPaymentAttemptSvc) FindByProviderLinkID(_ *zap.Logger, _ string) (PaymentAttempts, error) {
	return PaymentAttempts{}, nil
}

func (s stubPaymentAttemptSvc) FindByPaymentID(_ *zap.Logger, _ string) (PaymentAttempts, error) {
	return s.attempt, s.err
}

func (s stubPaymentAttemptSvc) FindByClientIdempotencyKey(_ *zap.Logger, _ string) (PaymentAttempts, error) {
	return PaymentAttempts{}, nil
}

func (s stubPaymentAttemptSvc) UpdatePaymentAttempt(_ *zap.Logger, _ *gorm.DB, _ PaymentAttempts) error {
	return nil
}

func (s stubPaymentAttemptSvc) UpdatePaymentAttemptStatus(_ *zap.Logger, _ *gorm.DB, _ PaymentAttempts) error {
	return nil
}

func (s stubPaymentAttemptSvc) ClaimForProcessing(_ *zap.Logger, _ string) (PaymentAttempts, bool, error) {
	return PaymentAttempts{}, false, nil
}

func TestValidateIdempotencyKey(t *testing.T) {
	if err := validateIdempotencyKey(""); err == nil {
		t.Fatal("expected error for empty key")
	}
	if err := validateIdempotencyKey("  "); err == nil {
		t.Fatal("expected error for whitespace key")
	}
	if err := validateIdempotencyKey("idem-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToPaymentModel(t *testing.T) {
	svc := &PaymentsService{}
	cmd := servicetest.ValidCreatePaymentCommand()

	got := svc.toPaymentModel(cmd)
	if got.ID == "" {
		t.Fatal("expected generated payment id")
	}
	if got.Amount != cmd.Amount || got.Channel != cmd.Channel || got.Source != cmd.Source {
		t.Fatalf("unexpected payment fields: %+v", got)
	}
	if got.Currency != cmd.Currency || got.InvoiceID != cmd.InvoiceID || got.PatientID != cmd.PatientID {
		t.Fatalf("unexpected payment fields: %+v", got)
	}
	if got.IdempotencyKey != cmd.IdempotencyKey || got.InitiatedBy != cmd.InitiatedBy {
		t.Fatalf("unexpected payment fields: %+v", got)
	}
}

func TestResponseFromExistingPaymentAttemptNotFound(t *testing.T) {
	svc := &PaymentsService{
		PaymentAttempt: stubPaymentAttemptSvc{err: gorm.ErrRecordNotFound},
	}
	resp, err := svc.responseFromExistingPayment(servicetest.NopLogger(), Payments{ID: "pay-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PaymentLinkID != "" || resp.PaymentURL != "" || resp.ReferenceID != "" {
		t.Fatalf("expected empty response, got %+v", resp)
	}
}
