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
	paymentAttempts := sPAttempts.toPaymentAModel(internalPaymentID, paymentResponse, providerName)
	return sPAttempts.PaymentAttemptRepo.CreatePaymentAttempts(tx, paymentAttempts)
}

func (sPAttempts *SPaymentAttempts) FindByProviderLinkID(providerLinkID string) (PaymentAttempts, error) {
	return sPAttempts.PaymentAttemptRepo.FindByProviderLinkID(providerLinkID)
}

func (sPAttempts *SPaymentAttempts) toPaymentAModel(internalPaymentID string, paymentResponse dto.CreatePaymentResponse, providerName string) PaymentAttempts {
	var payAttempts PaymentAttempts
	payAttempts.ID = uuid.NewString()
	payAttempts.AttemptNo = 1
	payAttempts.PaymentID = internalPaymentID
	payAttempts.ProviderLinkID = paymentResponse.PaymentLinkID
	payAttempts.PaymentLink = paymentResponse.PaymentURL
	payAttempts.Provider = providerName
	payAttempts.PaymentStatus = constants.StatusPending
	payAttempts.PaymentLinkStatus = constants.StatusPending
	payAttempts.PaymentLink = paymentResponse.PaymentURL
	payAttempts.CreatedAt = time.Now()
	if paymentResponse.RequestPayload != nil {
		payAttempts.ProviderRequest = datatypes.JSONMap(paymentResponse.RequestPayload)
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
