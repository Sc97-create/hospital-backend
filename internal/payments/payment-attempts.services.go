package payments

import (
	"errors"
	"hospital-backend/internal/payments/dto"
	"hospital-backend/pkg/constants"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SPaymentAttempts struct {
	PaymentAttemptRepo IPaymentAttempts
}

func NewPaymentAttempts(paymentARepo IPaymentAttempts) *SPaymentAttempts {
	return &SPaymentAttempts{PaymentAttemptRepo: paymentARepo}
}

func (sPAttempts *SPaymentAttempts) CreateAttempt(tx *gorm.DB, internalPaymentID string, paymentResponse dto.CreatePaymentResponse, providerName string) error {
	count, err := sPAttempts.PaymentAttemptRepo.CountByPaymentID(internalPaymentID)
	if err != nil {
		return err
	}
	paymentAttempts := sPAttempts.toPaymentAModel(internalPaymentID, paymentResponse, providerName, int(count)+1, "")
	return sPAttempts.PaymentAttemptRepo.CreatePaymentAttempts(tx, paymentAttempts)
}

func (sPAttempts *SPaymentAttempts) CreateAttemptWithIdempotency(
	tx *gorm.DB,
	internalPaymentID string,
	paymentResponse dto.CreatePaymentResponse,
	providerName string,
	clientIdempotencyKey string,
) error {
	count, err := sPAttempts.PaymentAttemptRepo.CountByPaymentID(internalPaymentID)
	if err != nil {
		return err
	}
	paymentAttempts := sPAttempts.toPaymentAModel(internalPaymentID, paymentResponse, providerName, int(count)+1, clientIdempotencyKey)
	return sPAttempts.PaymentAttemptRepo.CreatePaymentAttempts(tx, paymentAttempts)
}

func (sPAttempts *SPaymentAttempts) FindByProviderLinkID(providerLinkID string) (PaymentAttempts, error) {
	return sPAttempts.PaymentAttemptRepo.FindByProviderLinkID(providerLinkID)
}

func (sPAttempts *SPaymentAttempts) FindByPaymentID(paymentID string) (PaymentAttempts, error) {
	return sPAttempts.PaymentAttemptRepo.FindByPaymentID(paymentID)
}

func (sPAttempts *SPaymentAttempts) FindByClientIdempotencyKey(key string) (PaymentAttempts, error) {
	return sPAttempts.PaymentAttemptRepo.FindByClientIdempotencyKey(key)
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
func (sPAttempts *SPaymentAttempts) UpdatePaymentAttempt(tx *gorm.DB, paymentAttempt PaymentAttempts) error {
	return sPAttempts.PaymentAttemptRepo.UpdatePaymentAttempt(tx, paymentAttempt)
}
func (sPAttempts *SPaymentAttempts) UpdatePaymentAttemptStatus(tx *gorm.DB, paymentAttempt PaymentAttempts) error {
	return sPAttempts.PaymentAttemptRepo.UpdatePaymentAttemptStatus(tx, paymentAttempt)
}

// ClaimForProcessing delegates atomic pending → processing transition to the repo
func (sPAttempts *SPaymentAttempts) ClaimForProcessing(providerLinkID string) (PaymentAttempts, bool, error) {
	paymentAttempt, claimed, err := sPAttempts.PaymentAttemptRepo.ClaimForProcessing(providerLinkID)
	if err != nil {
		return PaymentAttempts{}, false, err
	}
	if !claimed {
		return PaymentAttempts{}, false, errors.New("payment attempt already claimed")
	}
	return paymentAttempt, claimed, nil
}
