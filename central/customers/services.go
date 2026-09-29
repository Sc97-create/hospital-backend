package customers

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	dto "hospital-backend/central/customers/dto"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	bcryptCost              = 8
	signupCodeDigits        = 6
	centralOrgPlaceholderID = "00000000-0000-0000-0000-000000000000"
)

// AccessTokenIssuer issues short-lived access tokens (no refresh / no DB store).
type AccessTokenIssuer interface {
	AccessToken(userID, organisationID string) (string, error)
}

// NotificationEnqueuer enqueues email notifications.
type NotificationEnqueuer interface {
	Create(ctx context.Context, data notificationdto.CreateRequest) error
}

type CustomerService struct {
	Repo          CustomerRepository
	Tokens        AccessTokenIssuer
	Notifications NotificationEnqueuer
}

func NewCustomerService(repo CustomerRepository, tokens AccessTokenIssuer, notifications NotificationEnqueuer) *CustomerService {
	return &CustomerService{Repo: repo, Tokens: tokens, Notifications: notifications}
}

func (s *CustomerService) Signup(log *zap.Logger, payload dto.SignupPayload) (dto.SignupResult, error) {
	log = ensureLog(log)
	if err := validateSignupPayload(payload); err != nil {
		log.Warn("customer signup failed", zap.String("reason", "validation"), zap.Error(err))
		return dto.SignupResult{}, wrapError.ErrInvalidRequest
	}

	email := strings.ToLower(strings.TrimSpace(payload.WorkEmail))
	exists, err := s.Repo.ExistsByWorkEmail(log, email)
	if err != nil {
		log.Error("customer signup failed",
			zap.String("work_email", email),
			zap.String("reason", "email_lookup"),
			zap.Error(err),
		)
		return dto.SignupResult{}, wrapError.ErrCustomerCreateFailed
	}
	if exists {
		log.Warn("customer signup failed",
			zap.String("work_email", email),
			zap.String("reason", "email_exists"),
		)
		return dto.SignupResult{}, wrapError.ErrCustomerAlreadyExists
	}

	hash, err := hashPassword(payload.Password)
	if err != nil {
		log.Error("customer signup failed",
			zap.String("work_email", email),
			zap.String("reason", "hash_password"),
			zap.Error(err),
		)
		return dto.SignupResult{}, wrapError.ErrCustomerCreateFailed
	}

	customer := buildCustomer(payload.FullName, email, hash)
	if err := s.Repo.Create(log, customer); err != nil {
		log.Error("customer signup failed",
			zap.String("work_email", email),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return dto.SignupResult{}, wrapError.ErrCustomerCreateFailed
	}

	if err := s.issueAndSendVerificationCode(log, customer); err != nil {
		return dto.SignupResult{}, err
	}

	token, err := s.Tokens.AccessToken(customer.ID, "")
	if err != nil {
		log.Error("customer signup failed",
			zap.String("customer_id", customer.ID),
			zap.String("reason", "issue_token"),
			zap.Error(err),
		)
		return dto.SignupResult{}, wrapError.ErrCustomerCreateFailed
	}

	log.Info("customer signup success",
		zap.String("customer_id", customer.ID),
		zap.String("work_email", customer.WorkEmail),
	)
	return dto.SignupResult{
		CustomerID:  customer.ID,
		WorkEmail:   customer.WorkEmail,
		Status:      customer.Status,
		AccessToken: token,
	}, nil
}

func (s *CustomerService) VerifyEmail(log *zap.Logger, customerID string, payload dto.VerifyEmailPayload) error {
	log = ensureLog(log)
	customerID = strings.TrimSpace(customerID)
	code := strings.TrimSpace(payload.Code)
	if customerID == "" || code == "" || len(code) != signupCodeDigits {
		log.Warn("customer email verify failed", zap.String("reason", "validation"))
		return wrapError.ErrInvalidRequest
	}

	row, err := s.Repo.FindActiveSignupCode(log, customerID, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("customer email verify failed",
				zap.String("customer_id", customerID),
				zap.String("reason", "code_not_found"),
			)
			return wrapError.ErrInvalidVerificationCode
		}
		log.Error("customer email verify failed",
			zap.String("customer_id", customerID),
			zap.String("reason", "code_lookup"),
			zap.Error(err),
		)
		return wrapError.ErrCustomerVerifyFailed
	}
	if time.Now().After(row.ExpiresAt) {
		log.Warn("customer email verify failed",
			zap.String("customer_id", customerID),
			zap.String("reason", "code_expired"),
		)
		return wrapError.ErrVerificationCodeExpired
	}

	now := time.Now()
	if err := s.Repo.MarkSignupCodeVerified(log, row.ID, now); err != nil {
		log.Error("customer email verify failed",
			zap.String("customer_id", customerID),
			zap.String("reason", "mark_code"),
			zap.Error(err),
		)
		return wrapError.ErrCustomerVerifyFailed
	}
	if err := s.Repo.MarkEmailVerified(log, customerID, now); err != nil {
		log.Error("customer email verify failed",
			zap.String("customer_id", customerID),
			zap.String("reason", "mark_customer"),
			zap.Error(err),
		)
		return wrapError.ErrCustomerVerifyFailed
	}

	log.Info("customer email verify success", zap.String("customer_id", customerID))
	return nil
}

func (s *CustomerService) GetByTenantID(log *zap.Logger, tenantID string) (Customer, error) {
	log = ensureLog(log)
	customer, err := s.Repo.GetByTenantID(log, tenantID)
	if err != nil {
		if errors.Is(err, ErrCustomerNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("customer lookup failed", zap.String("tenant_id", tenantID), zap.String("reason", "not_found"))
			return Customer{}, wrapError.ErrCustomerNotFound
		}
		log.Error("customer lookup failed", zap.String("tenant_id", tenantID), zap.String("reason", "db_read"), zap.Error(err))
		return Customer{}, wrapError.ErrCustomerUpdateFailed
	}
	return customer, nil
}

func (s *CustomerService) issueAndSendVerificationCode(log *zap.Logger, customer Customer) error {
	plainCode, err := generateSignupCode()
	if err != nil {
		log.Error("customer signup failed",
			zap.String("customer_id", customer.ID),
			zap.String("reason", "code_generate"),
			zap.Error(err),
		)
		return wrapError.ErrCustomerCreateFailed
	}

	now := time.Now()
	row := SignupCode{
		ID:        uuid.NewString(),
		UserID:    customer.ID,
		Code:      plainCode,
		ExpiresAt: now.Add(time.Duration(constants.CustomerEmailVerifyExpiryMinutes) * time.Minute),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.Repo.CreateSignupCode(log, row); err != nil {
		log.Error("customer signup failed",
			zap.String("customer_id", customer.ID),
			zap.String("reason", "code_persist"),
			zap.Error(err),
		)
		return wrapError.ErrCustomerCreateFailed
	}

	if s.Notifications == nil {
		log.Error("customer signup failed",
			zap.String("customer_id", customer.ID),
			zap.String("reason", "notifications_unavailable"),
		)
		return wrapError.ErrCustomerCreateFailed
	}

	err = s.Notifications.Create(context.Background(), notificationdto.CreateRequest{
		NotificationType: constants.CustomerEmailVerificationEvent,
		Subject:          constants.CustomerEmailVerificationSubject,
		Data: map[string]interface{}{
			"employee_name":     customer.FullName,
			"employee_email":    customer.WorkEmail,
			"employee_id":       customer.ID,
			"organisation_id":   centralOrgPlaceholderID,
			"hospital_name":     "Hospital Portal",
			"verification_code": plainCode,
			"expiry_minutes":    fmt.Sprintf("%d", constants.CustomerEmailVerifyExpiryMinutes),
		},
	})
	if err != nil {
		log.Error("customer signup failed",
			zap.String("customer_id", customer.ID),
			zap.String("reason", "enqueue_email"),
			zap.Error(err),
		)
		return wrapError.ErrCustomerCreateFailed
	}
	return nil
}

func buildCustomer(fullName, email, passwordHash string) Customer {
	now := time.Now()
	return Customer{
		ID:           uuid.NewString(),
		FullName:     strings.TrimSpace(fullName),
		WorkEmail:    email,
		PasswordHash: passwordHash,
		Status:       constants.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func generateSignupCode() (string, error) {
	max := big.NewInt(1_000_000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
