package payments

import (
	"errors"
	"hospital-backend/internal/payments/dto"
	"hospital-backend/pkg/constants"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SPaymentAttempts struct {
	PaymentAttemptRepo IPaymentAttempts
}

func NewPaymentAttempts(paymentARepo IPaymentAttempts) *SPaymentAttempts {
	return &SPaymentAttempts{PaymentAttemptRepo: paymentARepo}
}

func (sPAttempts *SPaymentAttempts) CreateAttempt(
	log *zap.Logger,
	tx *gorm.DB,
	internalPaymentID string,
	paymentResponse dto.CreatePaymentResponse,
	providerName string,
) error {
	log = ensureLog(log)
	count, err := sPAttempts.PaymentAttemptRepo.CountByPaymentID(log, internalPaymentID)
	if err != nil {
		log.Error("payment attempt create failed",
			zap.String("payment_id", internalPaymentID),
			zap.String("reason", "count"),
			zap.Error(err),
		)
		return err
	}
	paymentAttempts := sPAttempts.toPaymentAModel(internalPaymentID, paymentResponse, providerName, int(count)+1, "")
	err = sPAttempts.PaymentAttemptRepo.CreatePaymentAttempts(log, tx, paymentAttempts)
	if err != nil {
		log.Error("payment attempt create failed",
			zap.String("payment_id", internalPaymentID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (sPAttempts *SPaymentAttempts) CreateAttemptWithIdempotency(
	log *zap.Logger,
	tx *gorm.DB,
	internalPaymentID string,
	paymentResponse dto.CreatePaymentResponse,
	providerName string,
	clientIdempotencyKey string,
) error {
	log = ensureLog(log)
	count, err := sPAttempts.PaymentAttemptRepo.CountByPaymentID(log, internalPaymentID)
	if err != nil {
		log.Error("payment attempt create failed",
			zap.String("payment_id", internalPaymentID),
			zap.String("reason", "count"),
			zap.Error(err),
		)
		return err
	}
	paymentAttempts := sPAttempts.toPaymentAModel(internalPaymentID, paymentResponse, providerName, int(count)+1, clientIdempotencyKey)
	err = sPAttempts.PaymentAttemptRepo.CreatePaymentAttempts(log, tx, paymentAttempts)
	if err != nil {
		log.Error("payment attempt create failed",
			zap.String("payment_id", internalPaymentID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (sPAttempts *SPaymentAttempts) FindByProviderLinkID(log *zap.Logger, providerLinkID string) (PaymentAttempts, error) {
	log = ensureLog(log)
	attempt, err := sPAttempts.PaymentAttemptRepo.FindByProviderLinkID(log, providerLinkID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error("payment attempt lookup failed",
			zap.String("provider_link_id", providerLinkID),
			zap.String("reason", "db_error"),
			zap.Error(err),
		)
	}
	return attempt, err
}

func (sPAttempts *SPaymentAttempts) FindByPaymentID(log *zap.Logger, paymentID string) (PaymentAttempts, error) {
	log = ensureLog(log)
	attempt, err := sPAttempts.PaymentAttemptRepo.FindByPaymentID(log, paymentID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error("payment attempt lookup failed",
			zap.String("payment_id", paymentID),
			zap.String("reason", "db_error"),
			zap.Error(err),
		)
	}
	return attempt, err
}

func (sPAttempts *SPaymentAttempts) FindByClientIdempotencyKey(log *zap.Logger, key string) (PaymentAttempts, error) {
	log = ensureLog(log)
	attempt, err := sPAttempts.PaymentAttemptRepo.FindByClientIdempotencyKey(log, key)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error("payment attempt lookup failed",
			zap.String("reason", "db_error"),
			zap.Error(err),
		)
	}
	return attempt, err
}

func (sPAttempts *SPaymentAttempts) toPaymentAModel(
	internalPaymentID string,
	paymentResponse dto.CreatePaymentResponse,
	providerName string,
	attemptNo int,
	clientIdempotencyKey string,
) PaymentAttempts {
	var payAttempts PaymentAttempts
	payAttempts.ID = uuid.NewString()
	payAttempts.AttemptNo = attemptNo
	payAttempts.PaymentID = internalPaymentID
	payAttempts.ProviderLinkID = paymentResponse.PaymentLinkID
	payAttempts.PaymentLink = paymentResponse.PaymentURL
	payAttempts.Provider = providerName
	payAttempts.PaymentStatus = constants.StatusPending
	payAttempts.PaymentLinkStatus = constants.StatusPending
	payAttempts.PaymentLink = paymentResponse.PaymentURL
	payAttempts.CreatedAt = time.Now()
	reqPayload := map[string]interface{}{}
	if paymentResponse.RequestPayload != nil {
		for k, v := range paymentResponse.RequestPayload {
			reqPayload[k] = v
		}
	}
	if clientIdempotencyKey != "" {
		reqPayload["client_idempotency_key"] = clientIdempotencyKey
	}
	if len(reqPayload) > 0 {
		payAttempts.ProviderRequest = datatypes.JSONMap(reqPayload)
	}
	return payAttempts
}

func (sPAttempts *SPaymentAttempts) UpdatePaymentAttempt(log *zap.Logger, tx *gorm.DB, paymentAttempt PaymentAttempts) error {
	log = ensureLog(log)
	err := sPAttempts.PaymentAttemptRepo.UpdatePaymentAttempt(log, tx, paymentAttempt)
	if err != nil {
		log.Error("payment attempt update failed",
			zap.String("payment_attempt_id", paymentAttempt.ID),
			zap.String("reason", "db_error"),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (sPAttempts *SPaymentAttempts) UpdatePaymentAttemptStatus(log *zap.Logger, tx *gorm.DB, paymentAttempt PaymentAttempts) error {
	log = ensureLog(log)
	err := sPAttempts.PaymentAttemptRepo.UpdatePaymentAttemptStatus(log, tx, paymentAttempt)
	if err != nil {
		log.Error("payment attempt status update failed",
			zap.String("payment_attempt_id", paymentAttempt.ID),
			zap.String("reason", "db_error"),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// ClaimForProcessing delegates atomic pending → processing. claimed=false means a concurrent
// webhook already owns it — callers should treat that as a successful no-op (duplicate).
func (sPAttempts *SPaymentAttempts) ClaimForProcessing(log *zap.Logger, providerLinkID string) (PaymentAttempts, bool, error) {
	log = ensureLog(log)
	paymentAttempt, claimed, err := sPAttempts.PaymentAttemptRepo.ClaimForProcessing(log, providerLinkID)
	if err != nil {
		log.Error("payment attempt claim failed",
			zap.String("provider_link_id", providerLinkID),
			zap.String("reason", "db_error"),
			zap.Error(err),
		)
		return PaymentAttempts{}, false, err
	}
	return paymentAttempt, claimed, nil
}
