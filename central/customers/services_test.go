package customers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"hospital-backend/central/customers"
	dto "hospital-backend/central/customers/dto"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type stubCustomerRepo struct {
	exists       bool
	existsErr    error
	createErr    error
	created      *customers.Customer
	codeErr      error
	createdCode  *customers.SignupCode
	findCode     customers.SignupCode
	findErr      error
	markCodeErr  error
	markEmailErr error
}

func (s *stubCustomerRepo) Create(_ *zap.Logger, customer customers.Customer) error {
	if s.createErr != nil {
		return s.createErr
	}
	cp := customer
	s.created = &cp
	return nil
}

func (s *stubCustomerRepo) ExistsByWorkEmail(_ *zap.Logger, _ string) (bool, error) {
	if s.existsErr != nil {
		return false, s.existsErr
	}
	return s.exists, nil
}

func (s *stubCustomerRepo) GetByID(_ *zap.Logger, _ string) (customers.Customer, error) {
	return customers.Customer{}, gorm.ErrRecordNotFound
}

func (s *stubCustomerRepo) GetByTenantID(_ *zap.Logger, _ string) (customers.Customer, error) {
	return customers.Customer{}, customers.ErrCustomerNotFound
}

func (s *stubCustomerRepo) MarkEmailVerified(_ *zap.Logger, _ string, _ time.Time) error {
	return s.markEmailErr
}

func (s *stubCustomerRepo) BindTenantID(_ *zap.Logger, _ *gorm.DB, _, _ string) error {
	return nil
}

func (s *stubCustomerRepo) CreateSignupCode(_ *zap.Logger, code customers.SignupCode) error {
	if s.codeErr != nil {
		return s.codeErr
	}
	cp := code
	s.createdCode = &cp
	return nil
}

func (s *stubCustomerRepo) FindActiveSignupCode(_ *zap.Logger, _, _ string) (customers.SignupCode, error) {
	if s.findErr != nil {
		return customers.SignupCode{}, s.findErr
	}
	return s.findCode, nil
}

func (s *stubCustomerRepo) MarkSignupCodeVerified(_ *zap.Logger, _ string, _ time.Time) error {
	return s.markCodeErr
}

type stubTokenIssuer struct {
	token string
	err   error
}

func (s *stubTokenIssuer) AccessToken(userID, _ string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.token == "" {
		return "access-" + userID, nil
	}
	return s.token, nil
}

type stubNotifier struct {
	err     error
	lastReq *notificationdto.CreateRequest
}

func (s *stubNotifier) Create(_ context.Context, data notificationdto.CreateRequest) error {
	if s.err != nil {
		return s.err
	}
	cp := data
	s.lastReq = &cp
	return nil
}

func TestServiceSignup(t *testing.T) {
	log := servicetest.NopLogger()

	tests := []struct {
		name    string
		payload dto.SignupPayload
		repo    *stubCustomerRepo
		tokens  *stubTokenIssuer
		notify  *stubNotifier
		wantErr error
	}{
		{
			name:    "missing full_name",
			payload: dto.SignupPayload{WorkEmail: "a@b.com", Password: "secret12"},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "email already exists",
			payload: dto.SignupPayload{FullName: "Ada", WorkEmail: "ada@acme.com", Password: "secret12"},
			repo:    &stubCustomerRepo{exists: true},
			wantErr: wrapError.ErrCustomerAlreadyExists,
		},
		{
			name:    "notify fails",
			payload: dto.SignupPayload{FullName: "Ada", WorkEmail: "ada@acme.com", Password: "secret12"},
			repo:    &stubCustomerRepo{},
			notify:  &stubNotifier{err: errors.New("smtp down")},
			wantErr: wrapError.ErrCustomerCreateFailed,
		},
		{
			name:    "success",
			payload: dto.SignupPayload{FullName: "Ada Lovelace", WorkEmail: "Ada@Acme.com", Password: "secret12"},
			repo:    &stubCustomerRepo{},
			tokens:  &stubTokenIssuer{},
			notify:  &stubNotifier{},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			if repo == nil {
				repo = &stubCustomerRepo{}
			}
			tokens := tt.tokens
			if tokens == nil {
				tokens = &stubTokenIssuer{}
			}
			notify := tt.notify
			if notify == nil {
				notify = &stubNotifier{}
			}
			svc := customers.NewCustomerService(repo, tokens, notify)
			got, err := svc.Signup(log, tt.payload)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err=%v want=%v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.AccessToken == "" || got.WorkEmail != "ada@acme.com" {
				t.Fatalf("unexpected result: %+v", got)
			}
			if repo.createdCode == nil || len(repo.createdCode.Code) != 6 {
				t.Fatalf("expected 6-digit code, got %+v", repo.createdCode)
			}
			if notify.lastReq == nil {
				t.Fatal("expected notification enqueue")
			}
			data := notify.lastReq.Data.(map[string]interface{})
			if data["employee_id"] != repo.created.ID {
				t.Fatalf("employee_id=%v want customer id", data["employee_id"])
			}
			if err := bcrypt.CompareHashAndPassword([]byte(repo.created.PasswordHash), []byte("secret12")); err != nil {
				t.Fatalf("password hash mismatch: %v", err)
			}
		})
	}
}

func TestServiceVerifyEmail(t *testing.T) {
	log := servicetest.NopLogger()
	now := time.Now()

	t.Run("invalid code length", func(t *testing.T) {
		svc := customers.NewCustomerService(&stubCustomerRepo{}, &stubTokenIssuer{}, &stubNotifier{})
		err := svc.VerifyEmail(log, "cust-1", dto.VerifyEmailPayload{Code: "12"})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("code not found", func(t *testing.T) {
		svc := customers.NewCustomerService(&stubCustomerRepo{findErr: gorm.ErrRecordNotFound}, &stubTokenIssuer{}, &stubNotifier{})
		err := svc.VerifyEmail(log, "cust-1", dto.VerifyEmailPayload{Code: "123456"})
		if !errors.Is(err, wrapError.ErrInvalidVerificationCode) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("expired", func(t *testing.T) {
		svc := customers.NewCustomerService(&stubCustomerRepo{
			findCode: customers.SignupCode{ID: "c1", ExpiresAt: now.Add(-time.Minute)},
		}, &stubTokenIssuer{}, &stubNotifier{})
		err := svc.VerifyEmail(log, "cust-1", dto.VerifyEmailPayload{Code: "123456"})
		if !errors.Is(err, wrapError.ErrVerificationCodeExpired) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		svc := customers.NewCustomerService(&stubCustomerRepo{
			findCode: customers.SignupCode{ID: "c1", ExpiresAt: now.Add(10 * time.Minute)},
		}, &stubTokenIssuer{}, &stubNotifier{})
		if err := svc.VerifyEmail(log, "cust-1", dto.VerifyEmailPayload{Code: "123456"}); err != nil {
			t.Fatalf("VerifyEmail: %v", err)
		}
	})
}
