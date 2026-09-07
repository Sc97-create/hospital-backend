package payments_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/payments"
	"hospital-backend/internal/payments/mocks"
	"hospital-backend/internal/testutil/servicetest"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func newPaymentsService(t *testing.T, repo payments.IPaymentsRepository, attempt payments.PaymentAttemptServicer) *payments.PaymentsService {
	t.Helper()
	return payments.NewPaymentsService(nil, repo, nil, attempt, nil, nil)
}

func TestServiceGetPaymentByIdempotencyKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()

	t.Run("empty key error", func(t *testing.T) {
		svc := newPaymentsService(t, nil, nil)
		_, err := svc.GetPaymentByIdempotencyKey(log, "  ")
		if err == nil {
			t.Fatal("expected error for empty idempotency key")
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewMockIPaymentsRepository(ctrl)
		want := payments.Payments{ID: "pay-1", IdempotencyKey: "idem-1"}
		repo.EXPECT().FindByIdempotencyKey(log, "idem-1").Return(want, nil)
		svc := newPaymentsService(t, repo, nil)
		got, err := svc.GetPaymentByIdempotencyKey(log, "idem-1")
		if err != nil || got.ID != want.ID {
			t.Fatalf("got %+v err=%v", got, err)
		}
	})
}

func TestServiceGetPaymentByInvoiceID(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()

	t.Run("empty id", func(t *testing.T) {
		svc := newPaymentsService(t, nil, nil)
		_, err := svc.GetPaymentByInvoiceID(log, "")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := mocks.NewMockIPaymentsRepository(ctrl)
		repo.EXPECT().FindByInvoiceID(log, "inv-missing").Return(payments.Payments{}, gorm.ErrRecordNotFound)
		svc := newPaymentsService(t, repo, nil)
		_, err := svc.GetPaymentByInvoiceID(log, "inv-missing")
		if !errors.Is(err, wrapError.ErrPaymentNotFound) {
			t.Fatalf("expected payment not found, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewMockIPaymentsRepository(ctrl)
		want := payments.Payments{ID: "pay-1", InvoiceID: "inv-1"}
		repo.EXPECT().FindByInvoiceID(log, "inv-1").Return(want, nil)
		svc := newPaymentsService(t, repo, nil)
		got, err := svc.GetPaymentByInvoiceID(log, "inv-1")
		if err != nil || got.ID != want.ID {
			t.Fatalf("got %+v err=%v", got, err)
		}
	})
}

func TestServiceGetPaymentURLByPaymentID(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()

	t.Run("not found returns empty string", func(t *testing.T) {
		attemptRepo := mocks.NewMockIPaymentAttempts(ctrl)
		attemptRepo.EXPECT().FindByPaymentID(log, "pay-missing").Return(payments.PaymentAttempts{}, gorm.ErrRecordNotFound)
		svc := newPaymentsService(t, nil, payments.NewPaymentAttempts(attemptRepo))
		url, err := svc.GetPaymentURLByPaymentID(log, "pay-missing")
		if err != nil || url != "" {
			t.Fatalf("expected empty url, got %q err=%v", url, err)
		}
	})

	t.Run("success returns link", func(t *testing.T) {
		attemptRepo := mocks.NewMockIPaymentAttempts(ctrl)
		attemptRepo.EXPECT().FindByPaymentID(log, "pay-1").Return(payments.PaymentAttempts{
			PaymentID:   "pay-1",
			PaymentLink: "https://pay.example/link",
		}, nil)
		svc := newPaymentsService(t, nil, payments.NewPaymentAttempts(attemptRepo))
		url, err := svc.GetPaymentURLByPaymentID(log, "pay-1")
		if err != nil || url != "https://pay.example/link" {
			t.Fatalf("got url=%q err=%v", url, err)
		}
	})
}

func TestServiceConfirmManualPayment(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()

	t.Run("missing invoice id", func(t *testing.T) {
		svc := newPaymentsService(t, nil, nil)
		err := svc.ConfirmManualPayment(log, "", "org-1", constants.PaymentCash, "")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("missing org id", func(t *testing.T) {
		svc := newPaymentsService(t, nil, nil)
		err := svc.ConfirmManualPayment(log, "inv-1", "", constants.PaymentCash, "")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("unsupported mode", func(t *testing.T) {
		svc := newPaymentsService(t, nil, nil)
		err := svc.ConfirmManualPayment(log, "inv-1", "org-1", "online", "")
		if !errors.Is(err, wrapError.ErrUnsupportedPaymentMode) {
			t.Fatalf("expected unsupported payment mode, got %v", err)
		}
	})

	t.Run("payment not found", func(t *testing.T) {
		repo := mocks.NewMockIPaymentsRepository(ctrl)
		repo.EXPECT().FindInvoiceByPaymentAttempt(log, gomock.Any(), "inv-1", "org-1").Return(payments.Payments{}, gorm.ErrRecordNotFound)
		svc := newPaymentsService(t, repo, nil)
		err := svc.ConfirmManualPayment(log, "inv-1", "org-1", constants.PaymentCash, "")
		if !errors.Is(err, wrapError.ErrPaymentConfirmFailed) {
			t.Fatalf("expected payment confirm failed, got %v", err)
		}
	})
}
