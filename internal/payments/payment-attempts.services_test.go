package payments_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/payments"
	"hospital-backend/internal/payments/dto"
	"hospital-backend/internal/payments/mocks"
	"hospital-backend/internal/testutil/servicetest"
	"hospital-backend/pkg/constants"

	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func newPaymentAttempts(t *testing.T, repo payments.IPaymentAttempts) *payments.SPaymentAttempts {
	t.Helper()
	return payments.NewPaymentAttempts(repo)
}

func TestSPaymentAttemptsCreateAttempt(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()
	resp := dto.CreatePaymentResponse{
		PaymentLinkID: "plink-1",
		PaymentURL:    "https://pay.example/link",
	}

	t.Run("count error", func(t *testing.T) {
		repo := mocks.NewMockIPaymentAttempts(ctrl)
		repo.EXPECT().CountByPaymentID(log, "pay-1").Return(int64(0), errors.New("count error"))
		svc := newPaymentAttempts(t, repo)
		if err := svc.CreateAttempt(log, nil, "pay-1", resp, constants.ProviderNameRazorpay); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("create error", func(t *testing.T) {
		repo := mocks.NewMockIPaymentAttempts(ctrl)
		repo.EXPECT().CountByPaymentID(log, "pay-1").Return(int64(0), nil)
		repo.EXPECT().CreatePaymentAttempts(log, gomock.Any(), gomock.Any()).Return(errors.New("create error"))
		svc := newPaymentAttempts(t, repo)
		if err := svc.CreateAttempt(log, nil, "pay-1", resp, constants.ProviderNameRazorpay); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success attempt no 2 when count=1", func(t *testing.T) {
		repo := mocks.NewMockIPaymentAttempts(ctrl)
		repo.EXPECT().CountByPaymentID(log, "pay-1").Return(int64(1), nil)
		repo.EXPECT().CreatePaymentAttempts(log, gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ interface{}, _ *gorm.DB, attempt payments.PaymentAttempts) error {
				if attempt.AttemptNo != 2 {
					t.Fatalf("expected attempt no 2, got %d", attempt.AttemptNo)
				}
				if attempt.PaymentID != "pay-1" || attempt.ProviderLinkID != "plink-1" {
					t.Fatalf("unexpected attempt fields: %+v", attempt)
				}
				return nil
			},
		)
		svc := newPaymentAttempts(t, repo)
		if err := svc.CreateAttempt(log, nil, "pay-1", resp, constants.ProviderNameRazorpay); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestSPaymentAttemptsCreateAttemptWithIdempotency(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()
	resp := dto.CreatePaymentResponse{PaymentLinkID: "plink-2", PaymentURL: "https://pay.example/retry"}

	repo := mocks.NewMockIPaymentAttempts(ctrl)
	repo.EXPECT().CountByPaymentID(log, "pay-1").Return(int64(0), nil)
	repo.EXPECT().CreatePaymentAttempts(log, gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ interface{}, _ *gorm.DB, attempt payments.PaymentAttempts) error {
			if attempt.AttemptNo != 1 {
				t.Fatalf("expected attempt no 1, got %d", attempt.AttemptNo)
			}
			if attempt.ProviderRequest["client_idempotency_key"] != "idem-retry" {
				t.Fatalf("expected idempotency key in provider request, got %+v", attempt.ProviderRequest)
			}
			return nil
		},
	)

	svc := newPaymentAttempts(t, repo)
	if err := svc.CreateAttemptWithIdempotency(log, nil, "pay-1", resp, constants.ProviderNameRazorpay, "idem-retry"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSPaymentAttemptsFindByProviderLinkID(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()
	want := payments.PaymentAttempts{ID: "attempt-1", ProviderLinkID: "plink-1"}

	t.Run("repo error", func(t *testing.T) {
		repo := mocks.NewMockIPaymentAttempts(ctrl)
		repo.EXPECT().FindByProviderLinkID(log, "plink-1").Return(payments.PaymentAttempts{}, errors.New("db error"))
		svc := newPaymentAttempts(t, repo)
		_, err := svc.FindByProviderLinkID(log, "plink-1")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewMockIPaymentAttempts(ctrl)
		repo.EXPECT().FindByProviderLinkID(log, "plink-1").Return(want, nil)
		svc := newPaymentAttempts(t, repo)
		got, err := svc.FindByProviderLinkID(log, "plink-1")
		if err != nil || got.ID != want.ID {
			t.Fatalf("got %+v err=%v", got, err)
		}
	})
}

func TestSPaymentAttemptsFindByPaymentID(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()
	want := payments.PaymentAttempts{ID: "attempt-1", PaymentID: "pay-1"}

	t.Run("repo error", func(t *testing.T) {
		repo := mocks.NewMockIPaymentAttempts(ctrl)
		repo.EXPECT().FindByPaymentID(log, "pay-1").Return(payments.PaymentAttempts{}, errors.New("db error"))
		svc := newPaymentAttempts(t, repo)
		_, err := svc.FindByPaymentID(log, "pay-1")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewMockIPaymentAttempts(ctrl)
		repo.EXPECT().FindByPaymentID(log, "pay-1").Return(want, nil)
		svc := newPaymentAttempts(t, repo)
		got, err := svc.FindByPaymentID(log, "pay-1")
		if err != nil || got.ID != want.ID {
			t.Fatalf("got %+v err=%v", got, err)
		}
	})
}

func TestSPaymentAttemptsFindByClientIdempotencyKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()
	want := payments.PaymentAttempts{ID: "attempt-1", PaymentID: "pay-1"}

	t.Run("repo error", func(t *testing.T) {
		repo := mocks.NewMockIPaymentAttempts(ctrl)
		repo.EXPECT().FindByClientIdempotencyKey(log, "idem-1").Return(payments.PaymentAttempts{}, errors.New("db error"))
		svc := newPaymentAttempts(t, repo)
		_, err := svc.FindByClientIdempotencyKey(log, "idem-1")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewMockIPaymentAttempts(ctrl)
		repo.EXPECT().FindByClientIdempotencyKey(log, "idem-1").Return(want, nil)
		svc := newPaymentAttempts(t, repo)
		got, err := svc.FindByClientIdempotencyKey(log, "idem-1")
		if err != nil || got.ID != want.ID {
			t.Fatalf("got %+v err=%v", got, err)
		}
	})
}

func TestSPaymentAttemptsUpdatePaymentAttemptStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()
	attempt := payments.PaymentAttempts{ID: "attempt-1", PaymentStatus: constants.StatusPending}

	t.Run("error", func(t *testing.T) {
		repo := mocks.NewMockIPaymentAttempts(ctrl)
		repo.EXPECT().UpdatePaymentAttemptStatus(log, gomock.Any(), attempt).Return(errors.New("update error"))
		svc := newPaymentAttempts(t, repo)
		if err := svc.UpdatePaymentAttemptStatus(log, nil, attempt); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewMockIPaymentAttempts(ctrl)
		repo.EXPECT().UpdatePaymentAttemptStatus(log, gomock.Any(), attempt).Return(nil)
		svc := newPaymentAttempts(t, repo)
		if err := svc.UpdatePaymentAttemptStatus(log, nil, attempt); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestSPaymentAttemptsClaimForProcessing(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()
	want := payments.PaymentAttempts{ID: "attempt-1", ProviderLinkID: "plink-1"}

	t.Run("error", func(t *testing.T) {
		repo := mocks.NewMockIPaymentAttempts(ctrl)
		repo.EXPECT().ClaimForProcessing(log, "plink-1").Return(payments.PaymentAttempts{}, false, errors.New("claim error"))
		svc := newPaymentAttempts(t, repo)
		_, claimed, err := svc.ClaimForProcessing(log, "plink-1")
		if err == nil || claimed {
			t.Fatalf("expected error without claim, got claimed=%v err=%v", claimed, err)
		}
	})

	t.Run("success claimed true", func(t *testing.T) {
		repo := mocks.NewMockIPaymentAttempts(ctrl)
		repo.EXPECT().ClaimForProcessing(log, "plink-1").Return(want, true, nil)
		svc := newPaymentAttempts(t, repo)
		got, claimed, err := svc.ClaimForProcessing(log, "plink-1")
		if err != nil || !claimed || got.ID != want.ID {
			t.Fatalf("got %+v claimed=%v err=%v", got, claimed, err)
		}
	})
}
